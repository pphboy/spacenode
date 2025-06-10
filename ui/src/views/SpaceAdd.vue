<template>
  <div class="space-add">
    <el-form label-width="120px">
      <el-form-item label="Select App">
        <el-select v-model="selectedApp" placeholder="Select an app">
          <el-option v-for="app in apps" :key="app.appid" :label="app.title" :value="app.appid">
            <div style="display: flex; align-items: center">
              <el-avatar :src="app.icon" style="margin-right: 10px" />
              <span>{{ app.title }}</span>
            </div>
          </el-option>
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="handleAdd">Add</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElLoading, ElMessage } from 'element-plus'

interface App {
  appid: string
  title: string
  icon: string
}

const apps = ref<App[]>([])
const selectedApp = ref('')

onMounted(async () => {
  try {
    const response = await axios.get('/api/lzcapp/applist')
    apps.value = response.data.apps
  } catch (error) {
    console.error('Failed to fetch apps:', error)
  }
})

const handleAdd = async () => {
  if (!selectedApp.value) return
  const loading = ElLoading.service({
    lock: true,
    text: 'Adding app...',
    background: 'rgba(0, 0, 0, 0.7)',
  })
  try {
    await axios.post('/api/app/add', null, {
      params: { appid: selectedApp.value }
    })
    loading.close()
    ElMessage.success('App added successfully')
  } catch (error) {
    console.error('Failed to add app:', error)
    ElMessage.error('Failed to add app')
  } finally {
    loading.close()
  }
}
</script>

<style scoped>
.space-add {
  padding: 20px;
}
</style>
