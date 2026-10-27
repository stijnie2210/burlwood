<script setup lang="ts">
import { computed } from 'vue';
import type * as graph from '../../../bindings/burlwood/internal/graph';

const props = defineProps<{ label: graph.Label }>();

const text = computed(() =>
  props.label.type === 'remote' && props.label.remote
    ? `${props.label.remote}/${props.label.name}`
    : props.label.name
);
</script>

<template>
  <span class="label" :class="`label--${label.type}`" :title="text">
    <svg
      v-if="label.type === 'head'"
      class="label-icon"
      viewBox="0 0 24 24"
      role="img"
      aria-label="Checked out"
    >
      <path d="M20 6L9 17l-5-5" />
    </svg>
    <svg
      v-else-if="label.type === 'remote'"
      class="label-icon"
      viewBox="0 0 24 24"
      role="img"
      aria-label="Remote"
    >
      <path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9z" />
    </svg>
    <svg
      v-else-if="label.type === 'tag'"
      class="label-icon"
      viewBox="0 0 24 24"
      role="img"
      aria-label="Tag"
    >
      <path d="M12 2H2v10l9.3 9.3a1 1 0 0 0 1.4 0l8.6-8.6a1 1 0 0 0 0-1.4z" />
      <circle cx="7" cy="7" r="1.5" />
    </svg>
    <span class="label-text">{{ text }}</span>
  </span>
</template>

<style src="../../assets/labels.css" />
