<template>
  <div 
    class="nextmeta-home-root"
    @mousemove="handleMouseMove"
    @mouseleave="handleMouseLeave"
    ref="containerRef"
  >
    <!-- Layer 1: Spline 官方 viewer 渲染的 3D 场景（与 spline.design 首页同款） -->
    <div class="spline-background-layer">
      <spline-viewer url="/NextMeta/scene.splinecode" loading-anim-type="spinner-small-dark"></spline-viewer>
    </div>

    <!-- Layer 2: Foreground Glassmorphism UI (Navbar and UI remain steady; 3D background reacts to interaction) -->
    <div class="parallax-viewport">
      <!-- Top Floating Glass Navigation Header - Fixed & Solid -->
      <header class="glass-navbar">
        <div class="nav-left">
          <div class="logo-orb">
            <span class="logo-core"></span>
          </div>
          <span class="brand-title">NextMeta</span>
          <span class="version-badge">v2.0.4</span>
        </div>

        <nav class="nav-links">
          <a href="/NextMeta/guide/" class="nav-link">文档指南</a>
          <a href="/NextMeta/部署指南.html" class="nav-link">部署指南</a>
          <a href="/NextMeta/功能说明.html" class="nav-link">功能说明</a>
          <a href="/NextMeta/功能说明.html#our-advantages" class="nav-link">我们的优势</a>
        </nav>

        <div class="nav-right">
          <a 
            href="https://github.com/Audi-dask/NextMeta" 
            target="_blank" 
            rel="noopener noreferrer"
            class="github-button glass-pill"
          >
            <svg class="icon-gh" viewBox="0 0 24 24" width="18" height="18" stroke="currentColor" stroke-width="2" fill="none">
              <path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-0.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77A5.07 5.07 0 0 0 19.91 1S18.73 0.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27 0.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"></path>
            </svg>
            <span>Audi-dask/NextMeta</span>
            <span class="star-count">★ Star</span>
          </a>
        </div>
      </header>

      <!-- Hero Section -->
      <main class="hero-container">
        <!-- Status Tag -->
        <div class="badge-wrapper">
          <div class="meta-badge glass-card">
            <span class="pulse-indicator"></span>
            <span class="badge-text">Open Source SQL Audit Platform</span>
          </div>
        </div>

        <!-- Main Title with Iridescent Glass Glow -->
        <h1 class="hero-title">
          <span class="title-shimmer">NextMeta</span>
        </h1>

        <!-- Subtitle -->
        <p class="hero-subtitle">数据库 SQL 审核平台</p>
        <p class="hero-description">
          轻量级、简约风的数据库变更管控平台。统一查询入口、工单审批流与审计追踪，
          让每一次 DDL / DML 变更都经过规范审核，安全落地。
        </p>

        <!-- CTA Buttons Group (Glassmorphism + Glow) -->
        <div class="cta-group">
          <a href="/NextMeta/guide/" class="btn-primary-glass">
            <span class="btn-text">快速开始</span>
            <span class="btn-arrow">➔</span>
            <div class="btn-glow-border"></div>
          </a>

          <a href="https://github.com/Audi-dask/NextMeta" target="_blank" rel="noopener noreferrer" class="btn-secondary-glass">
            <svg class="btn-icon" viewBox="0 0 24 24" width="18" height="18" stroke="currentColor" stroke-width="2" fill="none">
              <polygon points="5 3 19 12 5 21 5 3"></polygon>
            </svg>
            <span>GitHub 仓库</span>
          </a>

          <button @click="copyQuickInstall" class="btn-terminal-glass" title="点击复制一键安装命令">
            <span class="term-prefix">$</span>
            <code>curl -fsSL .../install.sh | bash</code>
            <span class="copy-status">{{ copied ? '已复制 ✓' : '复制' }}</span>
          </button>
        </div>

        <!-- Interactive 3D Holographic Schema Preview Card -->
        <div class="hero-showcase-glass glass-panel" :style="cardParallaxStyle">
          <div class="glass-reflection-line"></div>
          
          <div class="panel-header">
            <div class="window-controls">
              <span class="dot red"></span>
              <span class="dot yellow"></span>
              <span class="dot green"></span>
            </div>
            <div class="panel-title-tab">
              <span class="tab-icon">⚡</span>
              <span>install.sh — one-click deploy</span>
            </div>
            <span class="panel-tag">ONE-CLICK</span>
          </div>

          <div class="panel-body">
            <div class="code-preview">
              <span class="code-comment"># 1. 一键安装：自动拉取最新配置并启动服务</span><br/>
              <span class="code-keyword">curl</span> -fsSL https://raw.githubusercontent.com/Audi-dask/NextMeta/main/install.sh | <span class="code-keyword">bash</span><br/>
              <br/>
              <span class="code-comment"># 2. 访问控制台</span><br/>
              <span class="code-func">open</span> <span class="code-str">http://localhost:8080</span><br/>
              <br/>
              <span class="code-comment"># 默认管理员 NextMeta / password123，登录后请立即修改</span>
            </div>

            <!-- Live Engine Stats Subcard -->
            <div class="live-metrics-card glass-subcard">
              <div class="metric-item">
                <span class="metric-label">部署方式</span>
                <span class="metric-value font-mono">Docker Compose</span>
              </div>
              <div class="metric-divider"></div>
              <div class="metric-item">
                <span class="metric-label">数据库</span>
                <span class="metric-value font-mono">MySQL 8.0</span>
              </div>
              <div class="metric-divider"></div>
              <div class="metric-item">
                <span class="metric-label">技术栈</span>
                <span class="metric-value status-active">● Go + React</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 4-Pillar Feature Bento Grid -->
        <section class="feature-grid">
          <div 
            v-for="(feature, index) in features" 
            :key="index"
            class="feature-card glass-card"
            @mouseenter="activeFeature = index"
          >
            <div class="card-glow-bg"></div>
            <h3 class="feature-title">{{ feature.title }}</h3>
            <p class="feature-desc">{{ feature.desc }}</p>
            <div class="feature-pill">{{ feature.tag }}</div>
          </div>
        </section>
      </main>

      <!-- Footer -->
      <footer class="glass-footer">
        <div class="footer-content">
          <span>Released under the MIT License • NextMeta SQL Audit Platform</span>
          <span class="footer-copy">© {{ currentYear }} Audi-dask / NextMeta. All rights reserved.</span>
        </div>
      </footer>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue';

// ----------------------------------------------------
// 1. 状态与响应式变量
// ----------------------------------------------------
const containerRef = ref<HTMLElement | null>(null);
const currentYear = new Date().getFullYear();
const copied = ref(false);
const activeFeature = ref(0);

// 鼠标位置与平滑插值 (卡片视差)
const mouse = reactive({
  targetX: 0,
  targetY: 0,
  currentX: 0,
  currentY: 0,
});

// 特性卡片数据
const features = [
  {
    title: '统一查询入口',
    desc: '多数据源集中管理，SQL 查询窗口内置安全限制，只读库自动拦截写操作。',
    tag: 'Query Console'
  },
  {
    title: '工单审批流',
    desc: 'DDL / DML 变更以工单形式提交，多级审批后方可执行，杜绝随意改库。',
    tag: 'Ticket Workflow'
  },
  {
    title: '权限与审计',
    desc: '基于角色的访问控制，禁用即时生效；全链路操作审计，变更可追溯。',
    tag: 'RBAC & Audit'
  },
  {
    title: '一键 Docker 部署',
    desc: 'Go + React 技术栈，docker compose 一键拉起，内置 MySQL 初始化脚本。',
    tag: 'Self-Hosted'
  }
];

// ----------------------------------------------------
// 2. 视差动画样式计算 (Parallax Tilt)
// ----------------------------------------------------
const cardParallaxStyle = computed(() => {
  const moveX = mouse.currentX * 16;
  const moveY = mouse.currentY * 16;
  const rotateX = -mouse.currentY * 4;
  const rotateY = mouse.currentX * 5;
  return {
    transform: `translate3d(${moveX.toFixed(2)}px, ${moveY.toFixed(2)}px, 20px) rotateX(${rotateX.toFixed(2)}deg) rotateY(${rotateY.toFixed(2)}deg)`
  };
});

function handleMouseMove(e: MouseEvent) {
  if (!containerRef.value) return;
  const rect = containerRef.value.getBoundingClientRect();
  // 归一化到 [-1, 1] 区间
  mouse.targetX = ((e.clientX - rect.left) / rect.width) * 2 - 1;
  mouse.targetY = ((e.clientY - rect.top) / rect.height) * 2 - 1;
}

function handleMouseLeave() {
  mouse.targetX = 0;
  mouse.targetY = 0;
}

function copyQuickInstall() {
  navigator.clipboard.writeText('curl -fsSL https://raw.githubusercontent.com/Audi-dask/NextMeta/main/install.sh | bash');
  copied.value = true;
  setTimeout(() => {
    copied.value = false;
  }, 2000);
}

// ----------------------------------------------------
// 3. Spline 官方 viewer 加载（脚本与场景均自托管于 docs/public/spline）
// ----------------------------------------------------
function loadSplineViewer() {
  if (document.querySelector('#spline-viewer-script')) return;
  const script = document.createElement('script');
  script.id = 'spline-viewer-script';
  script.type = 'module';
  script.src = '/NextMeta/spline/spline-viewer.js';
  document.head.appendChild(script);
}

// 卡片视差插值循环
let rafId = 0;
function tick() {
  rafId = requestAnimationFrame(tick);
  mouse.currentX += (mouse.targetX - mouse.currentX) * 0.08;
  mouse.currentY += (mouse.targetY - mouse.currentY) * 0.08;
}

onMounted(() => {
  loadSplineViewer();
  tick();
});

onUnmounted(() => {
  cancelAnimationFrame(rafId);
});
</script>

<style scoped>
/* ====================================================
   NextMeta 现代 Glassmorphism 毛玻璃与 3D 空间样式
   ==================================================== */

.nextmeta-home-root {
  position: relative;
  min-height: 100vh;
  width: 100%;
  overflow-x: hidden;
  background-color: #070913;
  color: #f1f5f9;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  user-select: none;
}

/* 1. Spline 3D 场景背景层 */
.spline-background-layer {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  z-index: 1;
  overflow: hidden;
}

.spline-background-layer spline-viewer {
  display: block;
  width: 100%;
  height: 100%;
}

/* 2. 前景 3D 视差视口 (Foreground Parallax Viewport)
   容器整体不拦截指针，空白区域可拖拽旋转 3D 场景；交互元素单独恢复指针事件 */
.parallax-viewport {
  position: relative;
  z-index: 2;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  transform-style: preserve-3d;
  will-change: transform;
  pointer-events: none;
}

.glass-navbar,
.cta-group a,
.cta-group button,
.hero-showcase-glass,
.feature-card,
.glass-footer {
  pointer-events: auto;
}

/* 3. 顶部毛玻璃导航条 (Glass Navbar) */
.glass-navbar {
  position: sticky;
  top: 16px;
  margin: 0 auto;
  width: calc(100% - 48px);
  max-width: 1200px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  border-radius: 32px;
  background: rgba(15, 23, 42, 0.55);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  border: 1px solid rgba(255, 255, 255, 0.12);
  box-shadow: 0 16px 36px -10px rgba(0, 0, 0, 0.4);
  z-index: 50;
}

.nav-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.logo-orb {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: linear-gradient(135deg, #6366f1, #ec4899);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 0 15px rgba(99, 102, 241, 0.6);
}

.logo-core {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #ffffff;
}

.brand-title {
  font-size: 1.15rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  background: linear-gradient(to right, #ffffff, #cbd5e1);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.version-badge {
  font-size: 0.75rem;
  font-family: monospace;
  padding: 2px 8px;
  border-radius: 12px;
  background: rgba(99, 102, 241, 0.18);
  border: 1px solid rgba(99, 102, 241, 0.35);
  color: #a5b4fc;
}

.nav-links {
  display: flex;
  align-items: center;
  gap: 28px;
}

.nav-link {
  color: #94a3b8;
  text-decoration: none;
  font-size: 0.92rem;
  font-weight: 500;
  transition: all 0.2s ease;
}

.nav-link:hover {
  color: #ffffff;
  text-shadow: 0 0 10px rgba(255, 255, 255, 0.4);
}

.glass-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 18px;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #ffffff;
  text-decoration: none;
  font-size: 0.88rem;
  font-weight: 500;
  transition: all 0.25s ease;
}

.glass-pill:hover {
  background: rgba(255, 255, 255, 0.16);
  border-color: rgba(255, 255, 255, 0.3);
  transform: translateY(-1px);
}

.star-count {
  font-size: 0.78rem;
  padding: 2px 8px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.12);
  color: #fbbf24;
}

/* 4. 英雄区容器 (Hero Container) */
.hero-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 60px 24px 80px;
  max-width: 1200px;
  margin: 0 auto;
  width: 100%;
}

.badge-wrapper {
  margin-bottom: 24px;
}

.meta-badge {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 6px 18px;
  border-radius: 20px;
  background: rgba(15, 23, 42, 0.6);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(99, 102, 241, 0.3);
  box-shadow: 0 8px 24px -6px rgba(99, 102, 241, 0.2);
}

.pulse-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #10b981;
  box-shadow: 0 0 10px #10b981;
  animation: pulseDot 2s infinite;
}

@keyframes pulseDot {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(1.3); }
}

.badge-text {
  font-size: 0.85rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  background: linear-gradient(90deg, #a5b4fc, #38bdf8);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

/* 大标题 (NextMeta) */
.hero-title {
  font-size: clamp(3.5rem, 8vw, 6.5rem);
  font-weight: 900;
  line-height: 1.05;
  letter-spacing: -0.04em;
  margin: 0 0 16px;
  position: relative;
}

.title-shimmer {
  background: linear-gradient(135deg, #ffffff 0%, #cbd5e1 40%, #818cf8 80%, #38bdf8 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  filter: drop-shadow(0 0 35px rgba(99, 102, 241, 0.45));
}

.hero-subtitle {
  font-size: clamp(1.4rem, 3.2vw, 2.2rem);
  font-weight: 700;
  letter-spacing: -0.01em;
  line-height: 1.5;
  padding: 0.12em 0;
  margin: 0 0 20px;
  background: linear-gradient(to right, #f8fafc, #94a3b8);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.hero-description {
  max-width: 680px;
  font-size: 1.08rem;
  line-height: 1.7;
  color: #94a3b8;
  margin: 0 0 40px;
}

/* 5. CTA 交互按钮组 (CTA Group) */
.cta-group {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-bottom: 56px;
}

.btn-primary-glass {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 14px 32px;
  border-radius: 28px;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.85), rgba(139, 92, 246, 0.75));
  backdrop-filter: blur(16px);
  color: #ffffff;
  text-decoration: none;
  font-weight: 600;
  font-size: 1rem;
  box-shadow: 0 12px 30px -8px rgba(99, 102, 241, 0.5), inset 0 1px 1px rgba(255, 255, 255, 0.4);
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
  overflow: hidden;
}

.btn-primary-glass:hover {
  transform: translateY(-2px) scale(1.02);
  box-shadow: 0 18px 36px -6px rgba(99, 102, 241, 0.7), inset 0 1px 1px rgba(255, 255, 255, 0.6);
}

.btn-arrow {
  transition: transform 0.25s ease;
}

.btn-primary-glass:hover .btn-arrow {
  transform: translateX(4px);
}

.btn-secondary-glass {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 14px 28px;
  border-radius: 28px;
  background: rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #f1f5f9;
  text-decoration: none;
  font-weight: 600;
  font-size: 1rem;
  transition: all 0.25s ease;
}

.btn-secondary-glass:hover {
  background: rgba(255, 255, 255, 0.12);
  border-color: rgba(255, 255, 255, 0.3);
  transform: translateY(-2px);
}

.btn-terminal-glass {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 12px 22px;
  border-radius: 28px;
  background: rgba(15, 23, 42, 0.7);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(56, 189, 248, 0.25);
  color: #38bdf8;
  font-family: monospace;
  font-size: 0.92rem;
  cursor: pointer;
  transition: all 0.25s ease;
}

.btn-terminal-glass:hover {
  border-color: rgba(56, 189, 248, 0.6);
  box-shadow: 0 0 20px rgba(56, 189, 248, 0.25);
}

.copy-status {
  font-size: 0.75rem;
  background: rgba(56, 189, 248, 0.15);
  padding: 2px 8px;
  border-radius: 12px;
}

/* 6. 毛玻璃全息卡片展示 (Hero Showcase Panel) */
.hero-showcase-glass {
  width: 100%;
  max-width: 860px;
  border-radius: 20px;
  background: rgba(15, 23, 42, 0.55);
  backdrop-filter: blur(24px) saturate(190%);
  -webkit-backdrop-filter: blur(24px) saturate(190%);
  border: 1px solid rgba(255, 255, 255, 0.12);
  box-shadow: 0 30px 60px -15px rgba(0, 0, 0, 0.6), inset 0 1px 0 rgba(255, 255, 255, 0.2);
  margin-bottom: 64px;
  text-align: left;
  position: relative;
  overflow: hidden;
  transition: transform 0.15s ease-out;
}

.glass-reflection-line {
  position: absolute;
  top: 0;
  left: -100%;
  width: 200%;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.5), transparent);
  animation: shimmerLine 8s infinite;
}

@keyframes shimmerLine {
  0% { transform: translateX(0); }
  100% { transform: translateX(100%); }
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(255, 255, 255, 0.02);
}

.window-controls {
  display: flex;
  gap: 8px;
}

.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}
.dot.red { background: #ef4444; }
.dot.yellow { background: #f59e0b; }
.dot.green { background: #10b981; }

.panel-title-tab {
  font-size: 0.85rem;
  font-family: monospace;
  color: #cbd5e1;
  display: flex;
  align-items: center;
  gap: 6px;
}

.panel-tag {
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0.05em;
  padding: 2px 8px;
  border-radius: 6px;
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.panel-body {
  padding: 24px;
}

.code-preview {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.9rem;
  line-height: 1.7;
  color: #e2e8f0;
  margin-bottom: 20px;
}

.code-comment { color: #64748b; }
.code-keyword { color: #f43f5e; font-weight: 600; }
.code-func { color: #38bdf8; }
.code-str { color: #a5b4fc; }
.code-bool { color: #f59e0b; }

.live-metrics-card {
  display: flex;
  align-items: center;
  justify-content: space-around;
  padding: 14px 20px;
  border-radius: 12px;
  background: rgba(0, 0, 0, 0.35);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.metric-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.metric-label {
  font-size: 0.75rem;
  color: #94a3b8;
}

.metric-value {
  font-size: 0.95rem;
  font-weight: 600;
  color: #ffffff;
}

.status-active {
  color: #34d399;
  font-size: 0.85rem;
}

.metric-divider {
  width: 1px;
  height: 28px;
  background: rgba(255, 255, 255, 0.1);
}

/* 7. 特性卡片 Bento Grid (Feature Grid) */
.feature-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 20px;
  width: 100%;
  text-align: left;
}

.feature-card {
  position: relative;
  padding: 28px 24px;
  border-radius: 18px;
  background: rgba(15, 23, 42, 0.45);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: 0 10px 30px -10px rgba(0, 0, 0, 0.3);
  transition: all 0.35s cubic-bezier(0.16, 1, 0.3, 1);
  overflow: hidden;
}

.feature-card:hover {
  transform: translateY(-6px);
  border-color: rgba(99, 102, 241, 0.4);
  box-shadow: 0 20px 40px -10px rgba(99, 102, 241, 0.25);
  background: rgba(15, 23, 42, 0.65);
}

.feature-title {
  font-size: 1.15rem;
  font-weight: 700;
  margin: 0 0 10px;
  color: #f8fafc;
}

.feature-desc {
  font-size: 0.9rem;
  line-height: 1.6;
  color: #94a3b8;
  margin: 0 0 18px;
}

.feature-pill {
  display: inline-block;
  font-size: 0.72rem;
  font-family: monospace;
  padding: 3px 10px;
  border-radius: 10px;
  background: rgba(99, 102, 241, 0.12);
  border: 1px solid rgba(99, 102, 241, 0.25);
  color: #a5b4fc;
}

/* 8. 底部版权 (Footer) */
.glass-footer {
  margin-top: auto;
  width: 100%;
  padding: 24px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(7, 9, 19, 0.75);
  backdrop-filter: blur(12px);
}

.footer-content {
  max-width: 1200px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: center;
  font-size: 0.85rem;
  color: #64748b;
}

/* 响应式适配 */
@media (max-width: 768px) {
  .glass-navbar {
    width: calc(100% - 24px);
    padding: 0 16px;
  }
  .nav-links {
    display: none;
  }
  .hero-container {
    padding: 40px 16px 60px;
  }
  .cta-group {
    flex-direction: column;
    width: 100%;
  }
  .btn-primary-glass, .btn-secondary-glass, .btn-terminal-glass {
    width: 100%;
    justify-content: center;
  }
}
</style>
