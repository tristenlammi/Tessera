// Type declarations for the FileSystem API (used for folder upload via drag-drop)
// https://wicg.github.io/entries-api/

interface FileSystemEntry {
  readonly isFile: boolean
  readonly isDirectory: boolean
  readonly name: string
  readonly fullPath: string
  readonly filesystem: FileSystem
}

interface FileSystemFileEntry extends FileSystemEntry {
  readonly isFile: true
  readonly isDirectory: false
  file(successCallback: (file: File) => void, errorCallback?: (error: DOMException) => void): void
}

interface FileSystemDirectoryEntry extends FileSystemEntry {
  readonly isFile: false
  readonly isDirectory: true
  createReader(): FileSystemDirectoryReader
  getFile(
    path?: string | null,
    options?: FileSystemFlags,
    successCallback?: (entry: FileSystemFileEntry) => void,
    errorCallback?: (error: DOMException) => void
  ): void
  getDirectory(
    path?: string | null,
    options?: FileSystemFlags,
    successCallback?: (entry: FileSystemDirectoryEntry) => void,
    errorCallback?: (error: DOMException) => void
  ): void
}

interface FileSystemDirectoryReader {
  readEntries(
    successCallback: (entries: FileSystemEntry[]) => void,
    errorCallback?: (error: DOMException) => void
  ): void
}

interface FileSystemFlags {
  create?: boolean
  exclusive?: boolean
}

interface FileSystem {
  readonly name: string
  readonly root: FileSystemDirectoryEntry
}

// Extend DataTransferItem with webkitGetAsEntry
interface DataTransferItem {
  webkitGetAsEntry(): FileSystemEntry | null
}

// Extend HTMLInputElement for webkitdirectory attribute
interface HTMLInputElement {
  webkitdirectory: boolean
  directory: boolean
}

// Extend File with webkitRelativePath
interface File {
  readonly webkitRelativePath: string
}
