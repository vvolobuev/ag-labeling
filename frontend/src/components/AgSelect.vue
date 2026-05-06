<template>
  <div ref="root" class="ag-selectx" :class="{ open: isOpen, up: dropUp, small: size === 'small' }">
    <button type="button" class="ag-selectx-btn" @click="toggle">
      <span>{{ selectedLabel }}</span>
      <span class="arrow">▾</span>
    </button>
    <div v-if="isOpen" class="ag-selectx-menu">
      <button
        v-for="opt in options"
        :key="String(opt.value)"
        type="button"
        class="ag-selectx-item"
        :class="{ active: String(opt.value) === currentValue }"
        @click="choose(opt.value)"
      >
        {{ opt.label }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  options: { type: Array, default: () => [] },
  size: { type: String, default: 'md' },
  buttonLabel: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue', 'change'])
const root = ref(null)
const isOpen = ref(false)
const dropUp = ref(false)

const currentValue = computed(() => String(props.modelValue))
const selectedLabel = computed(() => {
  if (String(props.buttonLabel || '').trim() !== '') return String(props.buttonLabel).trim()
  const found = props.options.find((o) => String(o.value) === currentValue.value)
  return found ? found.label : ''
})

function choose(v) {
  emit('update:modelValue', v)
  emit('change', v)
  isOpen.value = false
}

function toggle() {
  if (!isOpen.value) {
    const el = root.value
    if (el) {
      const r = el.getBoundingClientRect()
      const viewH = window.innerHeight || document.documentElement.clientHeight || 0
      const estimatedMenuH = Math.min(280, Math.max(44, props.options.length * 38))
      const spaceBelow = viewH - r.bottom
      const spaceAbove = r.top
      dropUp.value = spaceBelow < estimatedMenuH && spaceAbove > spaceBelow
    }
  }
  isOpen.value = !isOpen.value
}

function onDocClick(e) {
  if (!root.value) return
  if (!root.value.contains(e.target)) isOpen.value = false
}

function onWindowBlur() {
  isOpen.value = false
}

onMounted(() => {
  document.addEventListener('pointerdown', onDocClick, true)
  window.addEventListener('blur', onWindowBlur)
  window.addEventListener('scroll', onWindowBlur, true)
})
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocClick, true)
  window.removeEventListener('blur', onWindowBlur)
  window.removeEventListener('scroll', onWindowBlur, true)
})
</script>

<style scoped>
.ag-selectx {
  position: relative;
  min-width: 120px;
  z-index: 60;
}
.ag-selectx-btn {
  width: 100%;
  min-height: 42px;
  border: none;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--ag-text);
  padding: 10px 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  cursor: pointer;
  font: inherit;
}
.ag-selectx.small .ag-selectx-btn {
  min-height: 30px;
  padding: 5px 8px;
  border-radius: 10px;
  font-size: 12px;
}
.arrow { color: var(--ag-muted); font-size: 12px; }
.ag-selectx.open .ag-selectx-btn {
  box-shadow: 0 0 0 2px rgba(77, 107, 254, 0.35);
}
.ag-selectx-menu {
  position: absolute;
  z-index: 9999;
  left: 0;
  right: 0;
  top: calc(100% + 6px);
  border-radius: 12px;
  border: 1px solid var(--ag-border);
  background: #242426;
  overflow: hidden;
}
.ag-selectx.up .ag-selectx-menu {
  top: auto;
  bottom: calc(100% + 6px);
}
.ag-selectx-item {
  width: 100%;
  border: none;
  text-align: left;
  background: transparent;
  color: var(--ag-text);
  padding: 9px 12px;
  cursor: pointer;
  font: inherit;
}
.ag-selectx-item:hover {
  background: rgba(77, 107, 254, 0.2);
}
.ag-selectx-item.active {
  background: rgba(77, 107, 254, 0.28);
  color: #fff;
}
</style>
