<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import * as pdfjsLib from 'pdfjs-dist'

// Set worker path
pdfjsLib.GlobalWorkerOptions.workerSrc = `https://cdnjs.cloudflare.com/ajax/libs/pdf.js/${pdfjsLib.version}/pdf.worker.min.js`

const props = defineProps<{
  url: string
  filename?: string
}>()

const emit = defineEmits<{
  close: []
}>()

interface Annotation {
  id: string
  pageNumber: number
  type: 'highlight' | 'underline' | 'comment' | 'drawing'
  content?: string
  position: { x: number; y: number; width?: number; height?: number }
  color: string
  createdAt: string
}

const containerRef = ref<HTMLDivElement>()
const canvasRef = ref<HTMLCanvasElement>()
const annotationCanvasRef = ref<HTMLCanvasElement>()
const pdfDoc = ref<pdfjsLib.PDFDocumentProxy | null>(null)
const currentPage = ref(1)
const totalPages = ref(0)
const scale = ref(1.2)
const loading = ref(true)
const error = ref<string | null>(null)

const annotations = ref<Annotation[]>([])
const selectedTool = ref<'select' | 'highlight' | 'underline' | 'comment' | 'draw' | 'sign'>('select')
const annotationColor = ref('#ffeb3b')
const showCommentInput = ref(false)
const commentPosition = ref({ x: 0, y: 0 })
const commentText = ref('')
const isDrawing = ref(false)
const drawingPath = ref<{ x: number; y: number }[]>([])

const colors = ['#ffeb3b', '#4caf50', '#2196f3', '#f44336', '#9c27b0', '#ff9800']

async function loadPDF() {
  loading.value = true
  error.value = null

  try {
    const loadingTask = pdfjsLib.getDocument(props.url)
    pdfDoc.value = await loadingTask.promise
    totalPages.value = pdfDoc.value.numPages
    await renderPage(currentPage.value)
  } catch (err: any) {
    console.error('Failed to load PDF:', err)
    error.value = err.message || 'Failed to load PDF'
  } finally {
    loading.value = false
  }
}

async function renderPage(pageNum: number) {
  if (!pdfDoc.value || !canvasRef.value) return

  try {
    const page = await pdfDoc.value.getPage(pageNum)
    const viewport = page.getViewport({ scale: scale.value })
    
    const canvas = canvasRef.value
    const context = canvas.getContext('2d')!
    
    canvas.height = viewport.height
    canvas.width = viewport.width

    // Also resize annotation canvas
    if (annotationCanvasRef.value) {
      annotationCanvasRef.value.height = viewport.height
      annotationCanvasRef.value.width = viewport.width
    }

    const renderContext = {
      canvasContext: context,
      viewport: viewport
    }

    await page.render(renderContext).promise
    renderAnnotations()
  } catch (err: any) {
    console.error('Failed to render page:', err)
    error.value = 'Failed to render page'
  }
}

function renderAnnotations() {
  if (!annotationCanvasRef.value) return

  const ctx = annotationCanvasRef.value.getContext('2d')!
  ctx.clearRect(0, 0, annotationCanvasRef.value.width, annotationCanvasRef.value.height)

  const pageAnnotations = annotations.value.filter(a => a.pageNumber === currentPage.value)

  pageAnnotations.forEach(annotation => {
    ctx.fillStyle = annotation.color
    ctx.strokeStyle = annotation.color
    ctx.globalAlpha = 0.4

    switch (annotation.type) {
      case 'highlight':
        ctx.fillRect(
          annotation.position.x,
          annotation.position.y,
          annotation.position.width || 100,
          annotation.position.height || 20
        )
        break
      case 'underline':
        ctx.lineWidth = 2
        ctx.beginPath()
        ctx.moveTo(annotation.position.x, annotation.position.y)
        ctx.lineTo(
          annotation.position.x + (annotation.position.width || 100),
          annotation.position.y
        )
        ctx.stroke()
        break
      case 'comment':
        ctx.globalAlpha = 1
        ctx.fillStyle = annotation.color
        ctx.beginPath()
        ctx.arc(annotation.position.x, annotation.position.y, 10, 0, 2 * Math.PI)
        ctx.fill()
        ctx.fillStyle = '#fff'
        ctx.font = 'bold 12px sans-serif'
        ctx.textAlign = 'center'
        ctx.textBaseline = 'middle'
        ctx.fillText('!', annotation.position.x, annotation.position.y)
        break
      case 'drawing':
        // Drawings stored as SVG path or points would be rendered here
        break
    }
    ctx.globalAlpha = 1
  })
}

function goToPage(page: number) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page
    renderPage(page)
  }
}

function previousPage() {
  goToPage(currentPage.value - 1)
}

function nextPage() {
  goToPage(currentPage.value + 1)
}

function zoomIn() {
  scale.value = Math.min(scale.value + 0.2, 3)
  renderPage(currentPage.value)
}

function zoomOut() {
  scale.value = Math.max(scale.value - 0.2, 0.5)
  renderPage(currentPage.value)
}

function fitWidth() {
  if (!containerRef.value || !pdfDoc.value) return
  // Reset scale to fit container width
  scale.value = 1.2
  renderPage(currentPage.value)
}

function handleCanvasClick(e: MouseEvent) {
  if (!annotationCanvasRef.value) return

  const rect = annotationCanvasRef.value.getBoundingClientRect()
  const x = e.clientX - rect.left
  const y = e.clientY - rect.top

  switch (selectedTool.value) {
    case 'highlight':
      addAnnotation({
        type: 'highlight',
        position: { x: x - 50, y: y - 10, width: 100, height: 20 }
      })
      break
    case 'underline':
      addAnnotation({
        type: 'underline',
        position: { x: x - 50, y: y, width: 100 }
      })
      break
    case 'comment':
      commentPosition.value = { x, y }
      showCommentInput.value = true
      break
  }
}

function handleMouseDown(e: MouseEvent) {
  if (selectedTool.value === 'draw') {
    isDrawing.value = true
    drawingPath.value = []
    const rect = annotationCanvasRef.value!.getBoundingClientRect()
    drawingPath.value.push({
      x: e.clientX - rect.left,
      y: e.clientY - rect.top
    })
  }
}

function handleMouseMove(e: MouseEvent) {
  if (!isDrawing.value || selectedTool.value !== 'draw') return

  const rect = annotationCanvasRef.value!.getBoundingClientRect()
  const x = e.clientX - rect.left
  const y = e.clientY - rect.top

  drawingPath.value.push({ x, y })

  // Draw in real-time
  const ctx = annotationCanvasRef.value!.getContext('2d')!
  ctx.strokeStyle = annotationColor.value
  ctx.lineWidth = 2
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'

  if (drawingPath.value.length >= 2) {
    const prev = drawingPath.value[drawingPath.value.length - 2]
    ctx.beginPath()
    ctx.moveTo(prev.x, prev.y)
    ctx.lineTo(x, y)
    ctx.stroke()
  }
}

function handleMouseUp() {
  if (isDrawing.value && drawingPath.value.length > 0) {
    isDrawing.value = false
    // Save drawing as annotation
    addAnnotation({
      type: 'drawing',
      position: { x: drawingPath.value[0].x, y: drawingPath.value[0].y }
    })
    drawingPath.value = []
  }
}

function addAnnotation(data: Partial<Annotation>) {
  const annotation: Annotation = {
    id: crypto.randomUUID(),
    pageNumber: currentPage.value,
    type: data.type!,
    position: data.position!,
    color: annotationColor.value,
    content: data.content,
    createdAt: new Date().toISOString()
  }

  annotations.value.push(annotation)
  renderAnnotations()
}

function addComment() {
  if (!commentText.value.trim()) return

  addAnnotation({
    type: 'comment',
    position: commentPosition.value,
    content: commentText.value
  })

  commentText.value = ''
  showCommentInput.value = false
}

function deleteAnnotation(id: string) {
  annotations.value = annotations.value.filter(a => a.id !== id)
  renderAnnotations()
}

function clearAnnotations() {
  annotations.value = annotations.value.filter(a => a.pageNumber !== currentPage.value)
  renderAnnotations()
}

async function downloadAnnotated() {
  // In a real implementation, this would merge annotations with PDF
  // For now, we'll just trigger download of original
  const link = document.createElement('a')
  link.href = props.url
  link.download = props.filename || 'document.pdf'
  link.click()
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'ArrowLeft') previousPage()
  else if (e.key === 'ArrowRight') nextPage()
  else if (e.key === 'Escape') emit('close')
  else if (e.key === '+' || e.key === '=') zoomIn()
  else if (e.key === '-') zoomOut()
}

watch(() => props.url, () => {
  loadPDF()
})

onMounted(() => {
  loadPDF()
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  if (pdfDoc.value) {
    pdfDoc.value.destroy()
  }
})
</script>

<template>
  <div class="fixed inset-0 bg-black/90 z-50 flex flex-col">
    <!-- Toolbar -->
    <div class="bg-gray-900 text-white px-4 py-2 flex items-center gap-4">
      <!-- Close button -->
      <button @click="emit('close')" class="p-2 hover:bg-gray-700 rounded">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>

      <!-- Filename -->
      <span class="text-sm font-medium truncate max-w-[200px]">{{ filename || 'PDF Document' }}</span>

      <div class="h-6 w-px bg-gray-600"></div>

      <!-- Navigation -->
      <div class="flex items-center gap-2">
        <button @click="previousPage" :disabled="currentPage <= 1" class="p-2 hover:bg-gray-700 rounded disabled:opacity-50">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <span class="text-sm">
          <input
            type="number"
            :value="currentPage"
            @change="goToPage(parseInt(($event.target as HTMLInputElement).value))"
            class="w-12 px-2 py-1 text-center bg-gray-800 border border-gray-600 rounded"
            min="1"
            :max="totalPages"
          />
          / {{ totalPages }}
        </span>
        <button @click="nextPage" :disabled="currentPage >= totalPages" class="p-2 hover:bg-gray-700 rounded disabled:opacity-50">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
        </button>
      </div>

      <div class="h-6 w-px bg-gray-600"></div>

      <!-- Zoom -->
      <div class="flex items-center gap-2">
        <button @click="zoomOut" class="p-2 hover:bg-gray-700 rounded">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 12H4" />
          </svg>
        </button>
        <span class="text-sm w-16 text-center">{{ Math.round(scale * 100) }}%</span>
        <button @click="zoomIn" class="p-2 hover:bg-gray-700 rounded">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
        </button>
        <button @click="fitWidth" class="p-2 hover:bg-gray-700 rounded" title="Fit width">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
          </svg>
        </button>
      </div>

      <div class="h-6 w-px bg-gray-600"></div>

      <!-- Annotation Tools -->
      <div class="flex items-center gap-1">
        <button
          @click="selectedTool = 'select'"
          :class="['p-2 rounded', selectedTool === 'select' ? 'bg-blue-600' : 'hover:bg-gray-700']"
          title="Select"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122" />
          </svg>
        </button>
        <button
          @click="selectedTool = 'highlight'"
          :class="['p-2 rounded', selectedTool === 'highlight' ? 'bg-blue-600' : 'hover:bg-gray-700']"
          title="Highlight"
        >
          <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
            <path d="M15.243 3.343l5.414 5.414-1.414 1.414-5.414-5.414 1.414-1.414zm-1.414 1.414L4.1 14.486l-.707 6.364 6.364-.707 9.728-9.728-5.657-5.657z" />
          </svg>
        </button>
        <button
          @click="selectedTool = 'underline'"
          :class="['p-2 rounded', selectedTool === 'underline' ? 'bg-blue-600' : 'hover:bg-gray-700']"
          title="Underline"
        >
          <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
            <path d="M5,21H19V19H5V21M12,17A6,6 0 0,0 18,11V3H15.5V11A3.5,3.5 0 0,1 12,14.5A3.5,3.5 0 0,1 8.5,11V3H6V11A6,6 0 0,0 12,17Z" />
          </svg>
        </button>
        <button
          @click="selectedTool = 'comment'"
          :class="['p-2 rounded', selectedTool === 'comment' ? 'bg-blue-600' : 'hover:bg-gray-700']"
          title="Add Comment"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
          </svg>
        </button>
        <button
          @click="selectedTool = 'draw'"
          :class="['p-2 rounded', selectedTool === 'draw' ? 'bg-blue-600' : 'hover:bg-gray-700']"
          title="Draw"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
          </svg>
        </button>

        <!-- Color picker -->
        <div class="flex items-center gap-1 ml-2">
          <button
            v-for="color in colors"
            :key="color"
            @click="annotationColor = color"
            :class="['w-6 h-6 rounded-full border-2', annotationColor === color ? 'border-white' : 'border-transparent']"
            :style="{ backgroundColor: color }"
          ></button>
        </div>
      </div>

      <div class="flex-1"></div>

      <!-- Actions -->
      <button @click="clearAnnotations" class="px-3 py-1.5 text-sm bg-gray-700 rounded hover:bg-gray-600">
        Clear Page
      </button>
      <button @click="downloadAnnotated" class="px-3 py-1.5 text-sm bg-blue-600 rounded hover:bg-blue-700">
        Download
      </button>
    </div>

    <!-- PDF Viewer -->
    <div ref="containerRef" class="flex-1 overflow-auto flex justify-center p-4 bg-gray-800">
      <!-- Loading state -->
      <div v-if="loading" class="flex items-center justify-center h-full">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-white"></div>
      </div>

      <!-- Error state -->
      <div v-else-if="error" class="flex items-center justify-center h-full">
        <div class="text-center">
          <svg class="w-16 h-16 mx-auto text-red-500 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <p class="text-white">{{ error }}</p>
        </div>
      </div>

      <!-- Canvas container -->
      <div v-else class="relative shadow-2xl">
        <canvas ref="canvasRef" class="bg-white"></canvas>
        <canvas
          ref="annotationCanvasRef"
          class="absolute top-0 left-0"
          :class="{ 'cursor-crosshair': selectedTool !== 'select' }"
          @click="handleCanvasClick"
          @mousedown="handleMouseDown"
          @mousemove="handleMouseMove"
          @mouseup="handleMouseUp"
          @mouseleave="handleMouseUp"
        ></canvas>

        <!-- Comment input popup -->
        <div
          v-if="showCommentInput"
          class="absolute bg-white rounded-lg shadow-xl p-3 w-64"
          :style="{ left: commentPosition.x + 'px', top: commentPosition.y + 'px' }"
        >
          <textarea
            v-model="commentText"
            rows="3"
            class="w-full px-2 py-1 border rounded text-sm resize-none"
            placeholder="Add a comment..."
            @keydown.enter.ctrl="addComment"
          ></textarea>
          <div class="flex justify-end gap-2 mt-2">
            <button
              @click="showCommentInput = false"
              class="px-2 py-1 text-xs text-gray-600 hover:bg-gray-100 rounded"
            >
              Cancel
            </button>
            <button
              @click="addComment"
              class="px-2 py-1 text-xs bg-blue-600 text-white rounded hover:bg-blue-700"
            >
              Add
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Annotations panel -->
    <div
      v-if="annotations.filter(a => a.pageNumber === currentPage).length > 0"
      class="absolute right-4 top-20 w-64 bg-white rounded-lg shadow-xl max-h-96 overflow-y-auto"
    >
      <div class="p-3 border-b font-medium text-gray-900">Annotations</div>
      <div class="p-2 space-y-2">
        <div
          v-for="annotation in annotations.filter(a => a.pageNumber === currentPage)"
          :key="annotation.id"
          class="p-2 bg-gray-50 rounded flex items-start gap-2"
        >
          <div
            class="w-4 h-4 rounded-full flex-shrink-0 mt-0.5"
            :style="{ backgroundColor: annotation.color }"
          ></div>
          <div class="flex-1 min-w-0">
            <div class="text-sm font-medium capitalize">{{ annotation.type }}</div>
            <div v-if="annotation.content" class="text-xs text-gray-600 truncate">
              {{ annotation.content }}
            </div>
          </div>
          <button
            @click="deleteAnnotation(annotation.id)"
            class="text-red-500 hover:text-red-700"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
