<template>
  <div v-show="isActive" :class="['tab-item', { active: isActive }]">
    <slot></slot>
  </div>
</template>

<script setup>
import { computed, inject, ref } from 'vue'

const props = defineProps({
  label: {
    type: String,
    required: true
  }
})

const activeTab = inject('activeTab')
const tabs = inject('tabs')

// 计算当前 TabItem 的索引
const currentIndex = computed(() => {
  if (tabs) {
    return tabs.findIndex(tab => tab.label === props.label)
  }
  return 0
})

const isActive = computed(() => {
  return activeTab?.value === currentIndex.value
})
</script>

<style scoped>
.tab-item {
  display: none;
}

.tab-item.active {
  display: block;
}
</style>

