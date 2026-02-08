package webdav

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/tessera/tessera/internal/repository"
	"github.com/tessera/tessera/internal/services"
	"github.com/tessera/tessera/internal/storage"
)

// Server handles WebDAV requests
type Server struct {
	fs          *FileSystem
	authService *services.AuthService
	fileService *services.FileService
	log         zerolog.Logger
}

// NewServer creates a new WebDAV server
func NewServer(fileRepo *repository.FileRepository, storage *storage.MinIOStorage, authService *services.AuthService, fileService *services.FileService, log zerolog.Logger) *Server {
	return &Server{
		fs:          NewFileSystem(fileRepo, storage, log),
		authService: authService,
		fileService: fileService,
		log:         log,
	}
}

// Handler returns a Fiber handler for WebDAV requests
func (s *Server) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()

		// OPTIONS must be handled without authentication so Windows WebClient
		// (and other clients) can discover DAV capabilities before sending credentials.
		if method == "OPTIONS" {
			return s.handleOptions(c)
		}

		// Authenticate via Basic Auth
		userID, err := s.authenticate(c)
		if err != nil {
			c.Set("WWW-Authenticate", `Basic realm="Tessera WebDAV"`)
			c.Set("DAV", "1, 2")
			c.Set("MS-Author-Via", "DAV")
			return c.Status(401).SendString("Unauthorized")
		}

		urlPath := c.Path()

		// Strip /webdav prefix
		urlPath = strings.TrimPrefix(urlPath, "/webdav")
		if urlPath == "" {
			urlPath = "/"
		}

		s.log.Debug().Str("method", method).Str("path", urlPath).Str("user", userID).Msg("WebDAV request")

		switch method {
		case "PROPFIND":
			return s.handlePropfind(c, userID, urlPath)
		case "GET", "HEAD":
			return s.handleGet(c, userID, urlPath)
		case "PUT":
			return s.handlePut(c, userID, urlPath)
		case "DELETE":
			return s.handleDelete(c, userID, urlPath)
		case "MKCOL":
			return s.handleMkcol(c, userID, urlPath)
		case "COPY":
			return s.handleCopy(c, userID, urlPath)
		case "MOVE":
			return s.handleMove(c, userID, urlPath)
		case "LOCK":
			return s.handleLock(c, userID, urlPath)
		case "UNLOCK":
			return s.handleUnlock(c, userID, urlPath)
		default:
			return c.Status(405).SendString("Method Not Allowed")
		}
	}
}

func (s *Server) authenticate(c *fiber.Ctx) (string, error) {
	auth := c.Get("Authorization")
	if auth == "" {
		return "", fmt.Errorf("no authorization header")
	}

	if !strings.HasPrefix(auth, "Basic ") {
		return "", fmt.Errorf("not basic auth")
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
	if err != nil {
		return "", err
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid credentials format")
	}

	email := parts[0]
	password := parts[1]

	// Authenticate user
	user, _, _, err := s.authService.Login(c.Context(), services.LoginInput{
		Email:     email,
		Password:  password,
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	})
	if err != nil {
		return "", err
	}

	return user.ID.String(), nil
}

func (s *Server) handleOptions(c *fiber.Ctx) error {
	c.Set("Allow", "OPTIONS, GET, HEAD, PUT, DELETE, PROPFIND, MKCOL, COPY, MOVE, LOCK, UNLOCK")
	c.Set("DAV", "1, 2")
	c.Set("MS-Author-Via", "DAV")
	return c.SendStatus(200)
}

func (s *Server) handlePropfind(c *fiber.Ctx, userID, urlPath string) error {
	depth := c.Get("Depth", "1")

	stat, err := s.fs.Stat(c.Context(), userID, urlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return c.Status(404).SendString("Not Found")
		}
		return c.Status(500).SendString(err.Error())
	}

	responses := []propfindResponse{}

	// Add the requested resource
	responses = append(responses, s.buildPropfindResponse(urlPath, stat))

	// If directory and depth > 0, add children
	if stat.IsDir() && depth != "0" {
		file, err := s.fs.OpenFile(c.Context(), userID, urlPath, os.O_RDONLY, 0)
		if err == nil {
			defer file.Close()
			children, _ := file.Readdir(-1)
			for _, child := range children {
				childPath := path.Join(urlPath, child.Name())
				responses = append(responses, s.buildPropfindResponse(childPath, child))
			}
		}
	}

	// Build XML response
	multiStatus := multistatus{
		Responses: responses,
	}

	xmlData, err := xml.MarshalIndent(multiStatus, "", "  ")
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	c.Set("Content-Type", "application/xml; charset=utf-8")
	return c.Status(207).Send(append([]byte(xml.Header), xmlData...))
}

func (s *Server) buildPropfindResponse(urlPath string, info os.FileInfo) propfindResponse {
	href := "/webdav" + urlPath
	if info.IsDir() && !strings.HasSuffix(href, "/") {
		href += "/"
	}

	props := propstat{
		Status: "HTTP/1.1 200 OK",
		Prop: prop{
			DisplayName:     info.Name(),
			GetLastModified: info.ModTime().UTC().Format(http.TimeFormat),
			CreationDate:    info.ModTime().UTC().Format(time.RFC3339),
		},
	}

	if info.IsDir() {
		props.Prop.ResourceType = &resourceType{Collection: &struct{}{}}
	} else {
		props.Prop.GetContentLength = info.Size()
		props.Prop.GetContentType = "application/octet-stream"
	}

	return propfindResponse{
		Href:     href,
		Propstat: props,
	}
}

func (s *Server) handleGet(c *fiber.Ctx, userID, urlPath string) error {
	file, err := s.fs.OpenFile(c.Context(), userID, urlPath, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return c.Status(404).SendString("Not Found")
		}
		return c.Status(500).SendString(err.Error())
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	if stat.IsDir() {
		return c.Status(405).SendString("Method Not Allowed")
	}

	c.Set("Content-Type", "application/octet-stream")
	c.Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	c.Set("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))

	if c.Method() == "HEAD" {
		return c.SendStatus(200)
	}

	// Stream the file
	_, err = io.Copy(c.Response().BodyWriter(), file)
	return err
}

func (s *Server) handlePut(c *fiber.Ctx, userID, urlPath string) error {
	urlPath = path.Clean("/" + urlPath)
	dir := path.Dir(urlPath)
	fileName := path.Base(urlPath)

	// Resolve parent directory
	var parentID string
	if dir != "/" && dir != "" {
		parent, err := s.fs.resolveFile(c.Context(), userID, dir)
		if err != nil {
			return c.Status(409).SendString("Parent directory not found")
		}
		parentID = parent.ID.String()
	}

	// Check if file exists (update) or new (create)
	existingFile, _ := s.fs.resolveFile(c.Context(), userID, urlPath)

	body := c.Body()
	reader := strings.NewReader(string(body))

	if existingFile != nil {
		// Update existing file
		_, err := s.fileService.UpdateFileContent(c.Context(), existingFile.ID.String(), userID, reader, int64(len(body)))
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		return c.SendStatus(204)
	}

	// Create new file
	_, err := s.fileService.Upload(c.Context(), userID, parentID, fileName, reader, int64(len(body)), "application/octet-stream")
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	return c.SendStatus(201)
}

func (s *Server) handleDelete(c *fiber.Ctx, userID, urlPath string) error {
	err := s.fs.RemoveAll(c.Context(), userID, urlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return c.Status(404).SendString("Not Found")
		}
		return c.Status(500).SendString(err.Error())
	}
	return c.SendStatus(204)
}

func (s *Server) handleMkcol(c *fiber.Ctx, userID, urlPath string) error {
	err := s.fs.Mkdir(c.Context(), userID, urlPath, 0755)
	if err != nil {
		if os.IsExist(err) {
			return c.Status(405).SendString("Already exists")
		}
		return c.Status(500).SendString(err.Error())
	}
	return c.SendStatus(201)
}

func (s *Server) handleCopy(c *fiber.Ctx, userID, urlPath string) error {
	dest := c.Get("Destination")
	if dest == "" {
		return c.Status(400).SendString("Destination header required")
	}

	// Parse destination path
	destPath := strings.TrimPrefix(dest, "http://"+c.Hostname())
	destPath = strings.TrimPrefix(destPath, "https://"+c.Hostname())
	destPath = strings.TrimPrefix(destPath, "/webdav")

	// Get source file
	srcFile, err := s.fs.resolveFile(c.Context(), userID, urlPath)
	if err != nil {
		return c.Status(404).SendString("Source not found")
	}

	// Resolve destination parent
	destDir := path.Dir(destPath)
	destName := path.Base(destPath)

	var destParentID string
	if destDir != "/" && destDir != "" {
		parent, err := s.fs.resolveFile(c.Context(), userID, destDir)
		if err != nil {
			return c.Status(409).SendString("Destination parent not found")
		}
		destParentID = parent.ID.String()
	}

	// Copy the file
	_, err = s.fileService.Copy(c.Context(), srcFile.ID.String(), userID, destParentID, destName)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	return c.SendStatus(201)
}

func (s *Server) handleMove(c *fiber.Ctx, userID, urlPath string) error {
	dest := c.Get("Destination")
	if dest == "" {
		return c.Status(400).SendString("Destination header required")
	}

	// Parse destination path
	destPath := strings.TrimPrefix(dest, "http://"+c.Hostname())
	destPath = strings.TrimPrefix(destPath, "https://"+c.Hostname())
	destPath = strings.TrimPrefix(destPath, "/webdav")

	err := s.fs.Rename(c.Context(), userID, urlPath, destPath)
	if err != nil {
		if os.IsNotExist(err) {
			return c.Status(404).SendString("Not Found")
		}
		return c.Status(500).SendString(err.Error())
	}

	return c.SendStatus(201)
}

func (s *Server) handleLock(c *fiber.Ctx, userID, urlPath string) error {
	// Simple lock implementation - always succeeds
	token := fmt.Sprintf("opaquelocktoken:%s", time.Now().UnixNano())

	lockDiscovery := lockDiscoveryResponse{
		ActiveLock: activeLock{
			LockType:  lockType{Write: &struct{}{}},
			LockScope: lockScope{Exclusive: &struct{}{}},
			Owner:     owner{Href: userID},
			Timeout:   "Second-3600",
			LockToken: lockTokenElem{Href: token},
			Depth:     "infinity",
		},
	}

	xmlData, _ := xml.MarshalIndent(lockDiscovery, "", "  ")

	c.Set("Content-Type", "application/xml; charset=utf-8")
	c.Set("Lock-Token", "<"+token+">")
	return c.Status(200).Send(append([]byte(xml.Header), xmlData...))
}

func (s *Server) handleUnlock(c *fiber.Ctx, userID, urlPath string) error {
	return c.SendStatus(204)
}

// XML structures for WebDAV responses
type multistatus struct {
	XMLName   xml.Name           `xml:"D:multistatus"`
	Xmlns     string             `xml:"xmlns:D,attr"`
	Responses []propfindResponse `xml:"D:response"`
}

func (m *multistatus) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "D:multistatus"
	start.Attr = []xml.Attr{{Name: xml.Name{Local: "xmlns:D"}, Value: "DAV:"}}
	return e.EncodeElement(struct {
		Responses []propfindResponse `xml:"D:response"`
	}{m.Responses}, start)
}

type propfindResponse struct {
	Href     string   `xml:"D:href"`
	Propstat propstat `xml:"D:propstat"`
}

type propstat struct {
	Prop   prop   `xml:"D:prop"`
	Status string `xml:"D:status"`
}

type prop struct {
	DisplayName      string        `xml:"D:displayname,omitempty"`
	GetContentLength int64         `xml:"D:getcontentlength,omitempty"`
	GetContentType   string        `xml:"D:getcontenttype,omitempty"`
	GetLastModified  string        `xml:"D:getlastmodified,omitempty"`
	CreationDate     string        `xml:"D:creationdate,omitempty"`
	ResourceType     *resourceType `xml:"D:resourcetype"`
}

type resourceType struct {
	Collection *struct{} `xml:"D:collection,omitempty"`
}

type lockDiscoveryResponse struct {
	XMLName    xml.Name   `xml:"D:prop"`
	Xmlns      string     `xml:"xmlns:D,attr"`
	ActiveLock activeLock `xml:"D:lockdiscovery>D:activelock"`
}

func (l *lockDiscoveryResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "D:prop"
	start.Attr = []xml.Attr{{Name: xml.Name{Local: "xmlns:D"}, Value: "DAV:"}}
	type alias lockDiscoveryResponse
	return e.EncodeElement((*alias)(l), start)
}

type activeLock struct {
	LockType  lockType      `xml:"D:locktype"`
	LockScope lockScope     `xml:"D:lockscope"`
	Owner     owner         `xml:"D:owner"`
	Timeout   string        `xml:"D:timeout"`
	LockToken lockTokenElem `xml:"D:locktoken"`
	Depth     string        `xml:"D:depth"`
}

type lockType struct {
	Write *struct{} `xml:"D:write,omitempty"`
}

type lockScope struct {
	Exclusive *struct{} `xml:"D:exclusive,omitempty"`
}

type owner struct {
	Href string `xml:"D:href"`
}

type lockTokenElem struct {
	Href string `xml:"D:href"`
}
