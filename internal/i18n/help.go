package i18n

// HelpFull is the key reference printed by --help.
var HelpFull = def(`List:
  ↑/↓, j/k               select a row (the top row is "New variable")
  enter                  edit the selected variable or comment
  a                      add a variable above the selected row (on the "New variable" row: after the last variable)
  r                      rename the selected variable
  n                      insert a blank line above the selected row
  c                      insert a comment above the selected row
  #                      comment out / restore the selected row
  y                      copy the selected row; a variable keeps a commented-out copy above it
  d                      delete the selected row (variables need two presses)
  f                      merge consecutive blank lines
  /                      search KEYs, values and comments
  u                      undo
  R                      reload the file
  v                      show / hide values
  q                      quit (changes are saved as you go)

Editing:
  enter                  confirm
  esc                    cancel
  ←/→, home/end          move the cursor
  ctrl/alt+←/→           jump by word
  backspace, delete      delete a character
  alt+backspace, ctrl+w  delete the previous word
  ctrl+u                 clear

New variable:
  =                      start the value right after the KEY
  ctrl+t                 toggle KEY upper-casing / keeping its case
  paste KEY=VALUE lines  import them all at once

Search:
  ↑/↓                    go to the previous / next match
  enter                  select the current match
`, `列表:
  ↑/↓, j/k                选择行（最上面一行是“新建变量”）
  enter                   编辑选中的变量或注释
  a                       新建变量，插在选中行上方（在“新建变量”行上则加到最后一个变量之后）
  r                       重命名选中的变量
  n                       在选中行上方插入空行
  c                       在选中行上方插入注释
  #                       注释 / 取消注释选中行
  y                       复制选中行；变量会在上方留一份注释备份
  d                       删除选中行（变量需连按两次）
  f                       合并连续空行
  /                       搜索 KEY、值和注释
  u                       撤销
  R                       重新加载文件
  v                       显示 / 隐藏值
  q                       退出（改动已实时保存）

编辑:
  enter                   确认
  esc                     取消
  ←/→, home/end           移动光标
  ctrl/alt+←/→            按单词跳转
  backspace, delete       删除字符
  alt+backspace, ctrl+w   删除前一个单词
  ctrl+u                  清空

新建变量时:
  =                       在 KEY 后直接开始输入值
  ctrl+t                  切换 KEY 自动大写 / 保留大小写
  粘贴多行 KEY=VALUE      批量导入变量

搜索时:
  ↑/↓                     跳到上一个 / 下一个匹配
  enter                   选中当前匹配
`)
