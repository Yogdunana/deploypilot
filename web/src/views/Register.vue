<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <el-icon :size="40" color="#409eff"><Promotion /></el-icon>
        <h2>DeployPilot</h2>
        <p>Create a new account</p>
      </div>
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        @submit.prevent="handleRegister"
      >
        <el-form-item label="Username" prop="username">
          <el-input
            v-model="form.username"
            placeholder="Enter username"
            prefix-icon="User"
            size="large"
          />
        </el-form-item>
        <el-form-item label="Password" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="Enter password"
            prefix-icon="Lock"
            size="large"
            show-password
          />
        </el-form-item>
        <el-form-item label="Confirm Password" prop="confirmPassword">
          <el-input
            v-model="form.confirmPassword"
            type="password"
            placeholder="Confirm password"
            prefix-icon="Lock"
            size="large"
            show-password
            @keyup.enter="handleRegister"
          />
        </el-form-item>
        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            style="width: 100%"
            @click="handleRegister"
          >
            Register
          </el-button>
        </el-form-item>
      </el-form>
      <div class="login-footer">
        <span>Already have an account?</span>
        <router-link to="/login">Sign in</router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api'
import { ElMessage } from 'element-plus'

const router = useRouter()
const formRef = ref(null)
const loading = ref(false)

const form = reactive({
  username: '',
  password: '',
  confirmPassword: '',
})

const validateConfirm = (rule, value, callback) => {
  if (value !== form.password) {
    callback(new Error('Passwords do not match'))
  } else {
    callback()
  }
}

const rules = {
  username: [{ required: true, message: 'Please enter username', trigger: 'blur' }],
  password: [
    { required: true, message: 'Please enter password', trigger: 'blur' },
    { min: 6, message: 'Password must be at least 6 characters', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: 'Please confirm password', trigger: 'blur' },
    { validator: validateConfirm, trigger: 'blur' },
  ],
}

async function handleRegister() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    await authApi.register({
      username: form.username,
      password: form.password,
    })
    ElMessage.success('Registration successful. Please login.')
    router.push('/login')
  } catch {
    // error handled by interceptor
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-card {
  width: 420px;
  padding: 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.login-header {
  text-align: center;
  margin-bottom: 30px;
}

.login-header h2 {
  margin: 12px 0 4px;
  color: #333;
}

.login-header p {
  color: #999;
  margin: 0;
}

.login-footer {
  text-align: center;
  color: #999;
  font-size: 14px;
}

.login-footer a {
  color: #409eff;
  text-decoration: none;
  margin-left: 4px;
}
</style>
