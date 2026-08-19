package web

import "embed"

/*
StaticFS 嵌入前端构建产物 dist 目录。
router 会基于该文件系统提供静态资源访问和 SPA 路由回退。
*/
//go:embed dist
var StaticFS embed.FS
