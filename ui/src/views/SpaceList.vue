<template>
  <div class="space-list">
    <el-table :data="nodes" style="width: 100%">
      <el-table-column label="NodeID" width="180">
        <template #default="{ row }">
          {{ row.node.node_id }}
        </template>
      </el-table-column>
      <el-table-column label="NodeType" width="180">
        <template #default="{ row }">
          {{ row.node.node_type }}
        </template>
      </el-table-column>
      <el-table-column label="AppID" width="180">
        <template #default="{ row }">
          {{ row.node.app_id }}
        </template>
      </el-table-column>
      <el-table-column prop="ip" label="IP" />
      <el-table-column label="Operation">
        <template #default="{ row }">
          <div v-if="row.node.node_type == 'app'">
            <el-button type="danger" size="small"
              @click="removeApp(row.node.node_id, row.node.app_id)">Remove</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

interface Node {
  id: string
  type: 'app' | 'client'
  status: string
}

const nodes = ref<Node[]>([])


async function refreshList() {
  try {
    const response = await axios.get('/api/space/list')
    nodes.value = response.data
  } catch (error) {
    console.error('Failed to fetch nodes:', error)
  }
}

onMounted(async () => {
  await refreshList()
})

async function removeApp(nodeid: string, appid?: string) {
  try {
    await axios.post(`/api/app/remove?nodeid=${nodeid}&appid=${appid}`)
    console.log('App removed successfully')
    // Optionally, refresh the list of nodes
    await refreshList()
    ElMessage.success(`${appid} App removed successfully`)
  } catch (err) {
    console.error(err)
    ElMessage.error('Failed to remove app')
  }
}

async function removeClient(appid: string) {
  //TODO: 
}
</script>

<style scoped>
.space-list {
  padding: 20px;
}
</style>
