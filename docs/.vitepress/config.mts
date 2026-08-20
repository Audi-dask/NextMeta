import { defineConfig } from 'vitepress'

// NextMeta 文档站配置。
// base 必须与 GitHub 仓库名一致，否则 GitHub Pages 子路径下静态资源会 404。
export default defineConfig({
  title: 'NextMeta',
  description: '轻量级数据库 SQL 审核平台文档',
  lang: 'zh-CN',
  base: '/NextMeta/',
  lastUpdated: true,
  // 存量文档中存在指向仓库 img/ 目录的相对图片链接，暂时跳过死链检查，后续迁移图片后再移除。
  ignoreDeadLinks: true,

  themeConfig: {
    nav: [
      { text: '指南', link: '/guide/' },
      { text: 'GitHub', link: 'https://github.com/Audi-dask/NextMeta' }
    ],

    sidebar: [
      {
        text: '开始使用',
        items: [
          { text: '简介', link: '/guide/' },
          { text: '部署指南', link: '/部署指南' }
        ]
      },
      {
        text: '功能文档',
        items: [
          { text: '功能说明', link: '/功能说明' },
          { text: '数据脱敏机制', link: '/脱敏规则测试文档' }
        ]
      },
      {
        text: '项目',
        items: [
          { text: '项目与社区', link: '/项目与社区' }
        ]
      }
    ],

    outline: { label: '本页目录' },
    docFooter: { prev: '上一篇', next: '下一篇' },
    lastUpdated: { text: '最后更新' },
    search: { provider: 'local' }
  }
})
