<script setup lang="ts">
import { ref } from 'vue'
import { Plus, Trash2, Check } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import type { OAuth2Client } from '@/api/modules/oauth2'

const props = defineProps<{
  open: boolean
  loading: boolean
  form: {
    name: string
    redirect_uris: string[]
    scopes: string[]
    grant_types: string[]
  }
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  submit: []
  'update:form': [form: typeof props.form]
}>()

const availableScopes = ['read', 'write', 'admin', 'apps', 'servers', 'deployments', 'monitor']
const availableGrantTypes = ['client_credentials', 'authorization_code']

const localForm = ref({ ...props.form, redirect_uris: [...props.form.redirect_uris], scopes: [...props.form.scopes], grant_types: [...props.form.grant_types] })

function addRedirectUri() {
  localForm.value.redirect_uris.push('')
}

function removeRedirectUri(index: number) {
  localForm.value.redirect_uris.splice(index, 1)
}

function handleSubmit() {
  emit('update:form', localForm.value)
  emit('submit')
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div class="bg-card rounded-lg shadow-xl w-full max-w-md p-6 border border-border">
        <h2 class="text-lg font-semibold text-foreground mb-4">Create OAuth2 Application</h2>
        <div class="space-y-4">
          <!-- Name -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1">Name</label>
            <input
              v-model="localForm.name"
              type="text"
              placeholder="Enter application name"
              class="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>

          <!-- Redirect URIs -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1">Redirect URIs</label>
            <div class="space-y-2">
              <div v-for="(uri, index) in localForm.redirect_uris" :key="index" class="flex items-center gap-2">
                <input
                  v-model="localForm.redirect_uris[index]"
                  type="text"
                  placeholder="https://example.com/callback"
                  class="flex-1 rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                />
                <button
                  v-if="localForm.redirect_uris.length > 1"
                  class="inline-flex items-center justify-center w-8 h-8 rounded-md text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors cursor-pointer"
                  @click="removeRedirectUri(index)"
                >
                  <Trash2 class="w-3.5 h-3.5" />
                </button>
              </div>
              <Button variant="outline" size="sm" @click="addRedirectUri">
                <template #icon>
                  <Plus class="w-3.5 h-3.5" />
                </template>
                Add URI
              </Button>
            </div>
          </div>

          <!-- Scopes -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-2">Scopes</label>
            <div class="flex flex-wrap gap-2">
              <label
                v-for="scope in availableScopes"
                :key="scope"
                class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-border text-xs cursor-pointer transition-colors"
                :class="localForm.scopes.includes(scope) ? 'bg-primary/10 border-primary text-primary' : 'bg-background text-muted-foreground hover:bg-accent'"
              >
                <input
                  type="checkbox"
                  :value="scope"
                  class="sr-only"
                  @change="(e: any) => {
                    if (e.target.checked) localForm.scopes.push(scope)
                    else localForm.scopes = localForm.scopes.filter(s => s !== scope)
                  }"
                />
                {{ scope }}
              </label>
            </div>
          </div>

          <!-- Grant Types -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-2">Grant Types</label>
            <div class="space-y-2">
              <label
                v-for="gt in availableGrantTypes"
                :key="gt"
                class="flex items-center gap-2 px-3 py-2 rounded-md border border-border text-sm cursor-pointer transition-colors"
                :class="localForm.grant_types.includes(gt) ? 'bg-primary/10 border-primary text-primary' : 'bg-background text-muted-foreground hover:bg-accent'"
              >
                <input
                  type="checkbox"
                  :value="gt"
                  class="sr-only"
                  @change="(e: any) => {
                    if (e.target.checked) localForm.grant_types.push(gt)
                    else localForm.grant_types = localForm.grant_types.filter(g => g !== gt)
                  }"
                />
                <span class="w-4 h-4 rounded border flex items-center justify-center shrink-0" :class="localForm.grant_types.includes(gt) ? 'bg-primary border-primary' : 'border-border'">
                  <Check v-if="localForm.grant_types.includes(gt)" class="w-3 h-3 text-primary-foreground" />
                </span>
                {{ gt === 'client_credentials' ? 'Client Credentials' : 'Authorization Code' }}
              </label>
            </div>
          </div>
        </div>

        <div class="flex justify-end gap-2 mt-6">
          <Button variant="outline" size="sm" @click="emit('update:open', false)">Cancel</Button>
          <Button size="sm" :loading="loading" :disabled="!localForm.name.trim()" @click="handleSubmit">
            <template v-if="!loading" #icon>
              <Plus class="w-3.5 h-3.5" />
            </template>
            Create
          </Button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
