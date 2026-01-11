<template>
  <div class="tabs-container">
    <div class="tabs-header">
      <button
        v-for="(tab, index) in tabs"
        :key="index"
        :class="['tab-button', { active: activeTab === index }]"
        @click="activeTab = index"
      >
        {{ tab.label }}
      </button>
    </div>
    <div class="tabs-content">
      <slot></slot>
    </div>
  </div>
</template>

<script setup>
import { ref, provide, useSlots } from 'vue'

const activeTab = ref(0)
const slots = useSlots()

// 获取所有 TabItem 的标签
const tabs = []
if (slots.default) {
  const children = slots.default()
  children.forEach((child) => {
    if (child.props?.label) {
      tabs.push({ label: child.props.label })
    }
  })
}

provide('activeTab', activeTab)
provide('tabs', tabs)
</script>

<style scoped>
.tabs-container {
  margin: 0.8rem 0;
  border-radius: 6px;
  border: 1px solid var(--vp-c-divider);
  overflow: hidden;
}

.tabs-header {
  display: flex;
  gap: 0;
  background-color: var(--vp-c-bg-soft);
  border-bottom: 1px solid var(--vp-c-divider);
  flex-wrap: wrap;
}

.tab-button {
  padding: 8px 12px;
  border: none;
  background: transparent;
  color: var(--vp-c-text-2);
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
  transition: all 0.2s ease;
}

.tab-button:hover {
  color: var(--vp-c-brand);
  background-color: var(--vp-c-bg-mute);
}

.tab-button.active {
  color: var(--vp-c-brand);
  border-bottom: 2px solid var(--vp-c-brand);
  background-color: var(--vp-c-bg);
}

.tabs-content {
  padding: 12px 14px;
  background-color: var(--vp-c-bg);
}
</style>

