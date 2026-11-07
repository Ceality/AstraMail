import { mkdirSync, writeFileSync } from 'node:fs'

// Vite 的 emptyOutDir 会清空 dist，这里重新放回占位文件，
// 保证 go:embed 在未构建前端时也能编译，且构建后 Git 工作区保持干净。
mkdirSync('dist', { recursive: true })
writeFileSync('dist/.gitkeep', '')
