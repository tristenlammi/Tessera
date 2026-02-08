import { describe, it, expect, vi } from 'vitest'
import { debounce } from '../debounce'

describe('debounce', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('delays function execution', () => {
    const fn = vi.fn()
    const debouncedFn = debounce(fn, 100)

    debouncedFn()
    
    expect(fn).not.toHaveBeenCalled()
    
    vi.advanceTimersByTime(100)
    
    expect(fn).toHaveBeenCalledTimes(1)
  })

  it('only executes once for multiple rapid calls', () => {
    const fn = vi.fn()
    const debouncedFn = debounce(fn, 100)

    debouncedFn()
    debouncedFn()
    debouncedFn()
    debouncedFn()
    
    vi.advanceTimersByTime(100)
    
    expect(fn).toHaveBeenCalledTimes(1)
  })

  it('resets timer on each call', () => {
    const fn = vi.fn()
    const debouncedFn = debounce(fn, 100)

    debouncedFn()
    vi.advanceTimersByTime(50)
    
    debouncedFn() // Reset timer
    vi.advanceTimersByTime(50)
    
    expect(fn).not.toHaveBeenCalled()
    
    vi.advanceTimersByTime(50)
    
    expect(fn).toHaveBeenCalledTimes(1)
  })

  it('passes arguments to the original function', () => {
    const fn = vi.fn()
    const debouncedFn = debounce(fn, 100)

    debouncedFn('arg1', 'arg2', 123)
    
    vi.advanceTimersByTime(100)
    
    expect(fn).toHaveBeenCalledWith('arg1', 'arg2', 123)
  })

  it('uses the last call arguments', () => {
    const fn = vi.fn()
    const debouncedFn = debounce(fn, 100)

    debouncedFn('first')
    debouncedFn('second')
    debouncedFn('third')
    
    vi.advanceTimersByTime(100)
    
    expect(fn).toHaveBeenCalledWith('third')
    expect(fn).toHaveBeenCalledTimes(1)
  })

  it('allows subsequent calls after delay', () => {
    const fn = vi.fn()
    const debouncedFn = debounce(fn, 100)

    debouncedFn('first')
    vi.advanceTimersByTime(100)
    
    expect(fn).toHaveBeenCalledTimes(1)
    expect(fn).toHaveBeenLastCalledWith('first')

    debouncedFn('second')
    vi.advanceTimersByTime(100)
    
    expect(fn).toHaveBeenCalledTimes(2)
    expect(fn).toHaveBeenLastCalledWith('second')
  })
})
