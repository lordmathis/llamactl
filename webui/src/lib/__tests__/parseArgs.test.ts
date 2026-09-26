import { describe, expect, it } from 'vitest'
import { parseArgs } from '@/lib/parseArgs'

describe('parseArgs', () => {
  it('splits on whitespace', () => {
    expect(parseArgs('--delay=30s --port 8053')).toEqual(['--delay=30s', '--port', '8053'])
  })

  it('collapses repeated whitespace', () => {
    expect(parseArgs('  serve   --port   {port}  ')).toEqual(['serve', '--port', '{port}'])
  })

  it('keeps quoted values as single arguments', () => {
    expect(parseArgs('--tensor-split "3,1"')).toEqual(['--tensor-split', '3,1'])
    expect(parseArgs("--msg 'hello world'")).toEqual(['--msg', 'hello world'])
  })

  it('handles quotes glued to the value', () => {
    expect(parseArgs('--split="1, 2"')).toEqual(['--split=1, 2'])
  })

  it('returns an empty array for empty or blank input', () => {
    expect(parseArgs('')).toEqual([])
    expect(parseArgs('   ')).toEqual([])
  })
})
