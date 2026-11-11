<script setup>
defineProps({ name: { type: String, required: true } })

const shapes = {
  library: ['M3 4.5h3v15H3zM9 4.5h3v15H9zM15 5.4l3-.8 3.8 14.4-3 .8z'],
  book: ['M5 5a2 2 0 0 1 2-2h11a1 1 0 0 1 1 1v16H7a2 2 0 1 1 0-4h12M5 5v13M9 7h6M9 10h4'],
  plus: ['M12 5v14M5 12h14'],
  refresh: ['M20 9a8 8 0 1 0 .2 5M20 4v5h-5'],
  right: ['m9 6 6 6-6 6'],
  left: ['m15 6-6 6 6 6'],
  close: ['m6 6 12 12M18 6 6 18'],
  trash: ['M4 7h16M9 7V4h6v3m3 0-1 13H7L6 7m4 4v5m4-5v5'],
  check: ['m5 12 4 4L19 6'],
  alert: ['M12 8v4m0 4h.01', 'M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z'],
  stack: ['M3 7 12 3l9 4-9 4-9-4Zm0 5 9 4 9-4M3 17l9 4 9-4'],
}
</script>

<template>
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65"
    stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <path v-for="path in shapes[name]" :key="path" :d="path" />
  </svg>
</template>
