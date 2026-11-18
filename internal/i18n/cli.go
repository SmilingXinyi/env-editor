package i18n

// Texts of the command line.
var (
	ErrPrefix = def("error:", "错误：")
	Usage     = def(`Usage: env-editor [-m] [file]

Edit a .env file interactively (default: ./.env). Every change is saved
immediately; press u to undo.

Options:
  -m, --mask           hide values on start (toggle with v while running)
  -h, --help           show this help

`, `用法：env-editor [-m] [文件]

交互式编辑 .env 文件（默认 ./.env）。每次改动都会立即保存，可按 u 撤销。

选项：
  -m, --mask           启动时隐藏值（运行中按 v 切换）
  -h, --help           显示帮助

`)
)
