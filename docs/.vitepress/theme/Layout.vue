<script setup>
import DefaultTheme from 'vitepress/theme'
import { useData } from 'vitepress'
import CustomHome from './components/CustomHome.vue'

const { frontmatter } = useData()
</script>

<template>
  <CustomHome v-if="frontmatter.layout === 'custom-home'" />
  <DefaultTheme.Layout v-else />
</template>
