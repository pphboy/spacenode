<template>
  <div class="app-container">
    <el-container>
      <el-aside width="200px">
        <el-menu router>
          <el-menu-item index="/space/list">
            <el-icon>
              <Menu />
            </el-icon>
            <span>Space List</span>
          </el-menu-item>
          <el-menu-item index="/space/add">
            <el-icon>
              <Plus />
            </el-icon>
            <span>Add App</span>
          </el-menu-item>
        </el-menu>
      </el-aside>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </div>
</template>

<script lang="ts" setup>
import { Menu, Plus } from '@element-plus/icons-vue'
</script>

<style>
.app-container {
  height: 100vh;
}
</style>
