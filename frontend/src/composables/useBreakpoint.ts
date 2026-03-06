import { ref, onMounted, onUnmounted } from 'vue'

/** Tailwind md breakpoint in pixels */
const MD_BREAKPOINT = 768

export function useBreakpoint() {
  const isMobile = ref(false)

  function check() {
    isMobile.value = window.innerWidth < MD_BREAKPOINT
  }

  onMounted(() => {
    check()
    window.addEventListener('resize', check)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', check)
  })

  return { isMobile }
}
