package i18n

// Status line messages of the list.
var (
	FileMissing    = def("File does not exist; it will be created when the first change is saved", "文件不存在，保存第一个变量时会自动创建")
	QuitHint       = def("Press q to quit (changes are already saved)", "按 q 退出（改动已实时保存）")
	NoBlankRuns    = def("No consecutive blank lines", "没有连续的空行")
	BlanksMerged   = def("Merged blank lines (%d removed)", "已合并连续空行，删除了 %d 行")
	MultilineHint  = def("Multi-line values cannot be edited here; press r to rename or d to delete", "多行值暂不支持编辑，可按 r 重命名或 d 删除")
	RenameOnlyVars = def("Only variables can be renamed", "只有变量可以重命名")
	Commented      = def("Commented out %s", "已注释 %s")
	Uncommented    = def("Restored %s", "已取消注释 %s")
	ConfirmDelete  = def("Delete %s? Press d again to confirm, any other key cancels", "确认删除 %s？再按一次 d 删除，按其他键取消")
	DeletedVar     = def("Deleted %s; press u to undo", "已删除 %s，按 u 撤销")
	Deleted        = def("Deleted; press u to undo", "已删除，按 u 撤销")
	CommentBackup  = def("Kept a commented-out copy above", "已在上方留下注释备份")
	Copied         = def("Copied to the next line", "已复制到下一行")
	BlankInserted  = def("Inserted a blank line", "已插入空行")
	NothingToUndo  = def("Nothing to undo", "没有可撤销的操作")
	Undone         = def("Undone", "已撤销")
	ReloadFailed   = def("Reload failed: %s", "重新加载失败：%s")
	Reloaded       = def("Reloaded the file", "已重新加载文件")
)

// Status line messages of editing and importing.
var (
	PasteSingleLine   = def("Only a single line fits here; to import several variables, paste into the KEY field of a new variable", "这里只能输入单行内容；要批量导入变量，请在新建变量的 KEY 处粘贴")
	ImportNone        = def("No variables to import in the pasted text", "粘贴的内容里没有可导入的新变量")
	ImportNoneSkipped = def("No new variables to import (%d already exist)", "没有可导入的新变量（%d 个同名变量已存在）")
	Imported          = def("Imported %d variable(s)", "已导入 %d 个变量")
	ImportedSkipped   = def("Imported %d variable(s), skipped %d that already exist", "已导入 %d 个变量，%d 个同名变量已跳过")
	Cancelled         = def("Cancelled", "已取消")
	NoMatch           = def("No matches", "没有匹配项")
	Unchanged         = def("No change", "未修改")
	Saved             = def("Saved %s", "已保存 %s")
	SavedComment      = def("Saved comment", "已保存注释")
	Renamed           = def("Renamed %s → %s", "已重命名 %s → %s")
	Added             = def("Added %s", "已添加 %s")
)

// Error messages.
var (
	ErrInvalidKey   = def("Invalid KEY: use letters, digits and _ . - only, and do not start with a digit", "KEY 无效：只能包含字母、数字、_ . -，且不能以数字开头")
	ErrDuplicateKey = def("A variable with this name already exists", "已存在同名变量")
	ErrNotEntry     = def("This line is not a variable, so this does not apply", "这一行不是变量，无法执行此操作")
	ErrNotCommented = def("This comment is not a commented-out variable, so it cannot be restored", "这条注释不是被注释掉的变量，无法恢复")
	ErrMultiline    = def("Not supported for multi-line values yet", "多行值暂不支持此操作")
	ErrUnquotable   = def(`The value has a single quote and also \ or $, so it cannot be quoted safely; adjust it`, `值里既有单引号，又有 \ 或 $，无法安全加引号，请调整内容`)
	ErrModified     = def("The file was changed by another program and this change was not saved. Press R to reload, then try again", "文件已被其他程序修改，本次改动未保存。按 R 重新加载后再试")
	SaveFailed      = def("Save failed: %s", "保存失败：%s")
)
