<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { createBookstoreApi } from './api/bookstore.js'
import Icon from './components/BookstoreIcon.vue'

const api = createBookstoreApi()
const shelves = ref([])
const activeId = ref('')
const activeShelf = computed(() => shelves.value.find((shelf) => shelf.id === activeId.value))
const shelvesLoading = ref(true)
const shelvesError = ref('')
let shelfRequest

const books = ref([])
const booksLoading = ref(false)
const booksError = ref('')
const page = ref(0)
const pageTokens = ref([''])
const nextPageToken = ref('')
let bookRequest

const dialog = ref(null)
const action = ref(null)
const form = ref({})
const formError = ref('')
const saving = ref(false)
const notice = ref('')
let noticeTimeout

const dialogTitle = computed(() => ({
  createShelf: '新建书架',
  createBook: '添加图书',
  deleteShelf: '删除书架',
  deleteBook: '删除图书',
})[action.value?.mode] ?? '')

const deleting = computed(() => action.value?.mode.startsWith('delete'))

function showNotice(message) {
  notice.value = message
  clearTimeout(noticeTimeout)
  noticeTimeout = setTimeout(() => { notice.value = '' }, 4500)
}

async function loadBooks(index = 0, tokens = ['']) {
  if (!activeId.value) return
  bookRequest?.abort()
  const request = new AbortController()
  bookRequest = request
  booksLoading.value = true
  booksError.value = ''

  try {
    const result = await api.listBooks(activeId.value, tokens[index], request.signal)
    if (request.signal.aborted) return
    books.value = result.books
    nextPageToken.value = result.nextPageToken
    pageTokens.value = tokens
    page.value = index
  } catch (error) {
    if (!request.signal.aborted) booksError.value = error.message
  } finally {
    if (bookRequest === request) booksLoading.value = false
  }
}

function selectShelf(id) {
  if (id === activeId.value) return
  bookRequest?.abort()
  activeId.value = id
  books.value = []
  booksError.value = ''
  booksLoading.value = false
  page.value = 0
  pageTokens.value = ['']
  nextPageToken.value = ''
  if (id) void loadBooks()
}

async function loadShelves() {
  shelfRequest?.abort()
  const request = new AbortController()
  shelfRequest = request
  shelvesLoading.value = true
  shelvesError.value = ''
  try {
    const list = await api.listShelves(request.signal)
    if (request.signal.aborted) return
    shelves.value = list
    if (!list.some((shelf) => shelf.id === activeId.value)) {
      selectShelf(list[0]?.id ?? '')
    } else {
      void loadBooks()
    }
  } catch (error) {
    if (!request.signal.aborted) shelvesError.value = error.message
  } finally {
    if (shelfRequest === request) shelvesLoading.value = false
  }
}

function previousPage() {
  if (page.value > 0) void loadBooks(page.value - 1, pageTokens.value)
}

function nextPage() {
  if (!nextPageToken.value) return
  const tokens = [...pageTokens.value.slice(0, page.value + 1), nextPageToken.value]
  void loadBooks(page.value + 1, tokens)
}

async function openDialog(mode, target) {
  action.value = { mode, target, shelfId: activeId.value }
  form.value = { theme: '', size: 500, title: '', author: '' }
  formError.value = ''
  await nextTick()
  dialog.value.showModal()
}

function closeDialog() {
  if (!saving.value) dialog.value.close()
}

async function submitAction() {
  if (saving.value || !action.value) return
  const { mode, target, shelfId } = action.value
  saving.value = true
  formError.value = ''
  let succeeded = false

  try {
    switch (mode) {
      case 'createShelf': {
        const shelf = await api.createShelf({ theme: form.value.theme, size: String(form.value.size) })
        shelves.value = [...shelves.value, shelf]
        selectShelf(shelf.id)
        showNotice('书架已创建')
        break
      }
      case 'createBook':
        await api.createBook(shelfId, { title: form.value.title, author: form.value.author })
        if (shelfId === activeId.value) void loadBooks()
        showNotice('图书已添加')
        break
      case 'deleteShelf': {
        await api.deleteShelf(target.id)
        shelves.value = shelves.value.filter((shelf) => shelf.id !== target.id)
        if (activeId.value === target.id) selectShelf(shelves.value[0]?.id ?? '')
        showNotice('书架已删除')
        break
      }
      case 'deleteBook':
        await api.deleteBook(shelfId, target.id)
        if (shelfId === activeId.value) void loadBooks()
        showNotice('图书已删除')
        break
    }
    succeeded = true
  } catch (error) {
    formError.value = error.message
  } finally {
    saving.value = false
    if (succeeded) closeDialog()
  }
}

onMounted(loadShelves)
onBeforeUnmount(() => {
  shelfRequest?.abort()
  bookRequest?.abort()
  clearTimeout(noticeTimeout)
})
</script>

<template>
  <header class="site-header">
    <div class="container topbar">
      <div class="brand">
        <span class="brand-icon"><Icon name="library" /></span>
        <span class="brand-name">BookStore<span>.</span></span>
      </div>
      <span class="header-label">藏书管理</span>
    </div>
  </header>

  <main class="container">
    <section class="page-intro" aria-labelledby="page-title">
      <div>
        <p class="eyebrow">我的藏书</p>
        <h1 id="page-title">书架与藏书</h1>
        <p class="intro-description">按书架整理藏书，找到每一本书的位置。</p>
      </div>
      <button class="button primary" type="button" :disabled="shelvesLoading"
        @click="openDialog('createShelf')">
        <Icon name="plus" />新建书架
      </button>
    </section>

    <div class="workspace">
      <aside class="panel shelf-panel" aria-labelledby="shelf-heading">
        <header class="panel-heading">
          <div class="heading-group">
            <Icon name="library" />
            <h2 id="shelf-heading">书架</h2>
            <span v-if="!shelvesLoading && !shelvesError" class="count">{{ shelves.length }}</span>
          </div>
          <button type="button" class="icon-button" title="刷新书架" aria-label="刷新书架"
            :disabled="shelvesLoading" @click="loadShelves">
            <Icon name="refresh" />
          </button>
        </header>

        <div class="shelf-content" :aria-busy="shelvesLoading">
          <div v-if="shelvesLoading && !shelves.length" class="shelf-placeholder" role="status">
            正在加载书架…
          </div>
          <div v-if="shelvesError" class="inline-error" role="alert">
            <strong>无法加载书架</strong>
            <p>{{ shelvesError }}</p>
            <button class="button secondary small" type="button" @click="loadShelves">重试</button>
          </div>
          <p v-if="!shelvesLoading && !shelvesError && !shelves.length" class="shelf-placeholder">
            还没有书架。创建一个书架来整理你的藏书。
          </p>
          <ul v-if="shelves.length" class="shelf-list">
            <li v-for="shelf in shelves" :key="shelf.id">
              <button type="button" class="shelf-item" :class="{ selected: shelf.id === activeId }"
                :aria-current="shelf.id === activeId ? 'true' : undefined"
                @click="selectShelf(shelf.id)">
                <span class="shelf-icon"><Icon name="library" /></span>
                <span class="shelf-label">
                  <strong>{{ shelf.theme }}</strong>
                  <small>容量 {{ shelf.size }} 册</small>
                </span>
                <Icon class="shelf-chevron" name="right" />
              </button>
            </li>
          </ul>
        </div>
      </aside>

      <section class="panel book-panel" aria-labelledby="book-heading">
        <template v-if="activeShelf">
          <header class="book-heading">
            <div class="book-heading-title">
              <span class="shelf-badge"><Icon name="library" /></span>
              <div>
                <p class="eyebrow">当前书架</p>
                <h2 id="book-heading">{{ activeShelf.theme }}</h2>
                <p class="shelf-details">容量 {{ activeShelf.size }} 册 <span>·</span> 书架 #{{ activeShelf.id }}</p>
              </div>
            </div>
            <div class="heading-actions">
              <button class="icon-button shelf-delete" type="button" title="删除当前书架"
                aria-label="删除当前书架" :disabled="shelvesLoading" @click="openDialog('deleteShelf', activeShelf)">
                <Icon name="trash" />
              </button>
              <button class="button primary" type="button" @click="openDialog('createBook')">
                <Icon name="plus" />添加图书
              </button>
            </div>
          </header>

          <div class="book-content" :aria-busy="booksLoading">
            <h3>书架中的图书</h3>
            <div v-if="booksError" class="inline-error book-error" role="alert">
              <div><strong>无法加载图书</strong><p>{{ booksError }}</p></div>
              <button type="button" class="button secondary small" @click="loadBooks()">重新加载</button>
            </div>
            <div v-if="booksLoading && !books.length" class="empty-state compact" role="status">
              <span class="empty-icon"><Icon name="book" /></span>
              <strong>正在加载图书…</strong>
            </div>
            <div v-else-if="!booksError && !books.length" class="empty-state compact">
              <span class="empty-icon"><Icon name="book" /></span>
              <strong>这个书架还没有图书</strong>
              <p>添加第一本书，开始整理你的藏书。</p>
              <button class="button secondary small" type="button" @click="openDialog('createBook')">
                <Icon name="plus" />添加第一本图书
              </button>
            </div>
            <div v-if="books.length" class="book-table">
              <table>
                <thead>
                  <tr><th scope="col">书名</th><th scope="col">作者</th><th scope="col" class="book-id">编号</th><th scope="col" class="row-actions">操作</th></tr>
                </thead>
                <tbody>
                  <tr v-for="(book, index) in books" :key="book.id">
                    <td>
                      <div class="book-title">
                        <span class="book-cover" :class="{ 'cover-alt': index % 2 }"><Icon name="book" /></span>
                        <strong>{{ book.title }}</strong>
                      </div>
                    </td>
                    <td class="author">{{ book.author }}</td>
                    <td class="book-id">#{{ book.id }}</td>
                    <td class="row-actions">
                      <button type="button" class="icon-button delete-book" title="删除图书"
                        :aria-label="'删除《' + book.title + '》'" @click="openDialog('deleteBook', book)">
                        <Icon name="trash" />
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <footer v-if="books.length || page > 0" class="pagination" aria-label="图书分页">
            <span role="status">第 {{ page + 1 }} 页<span class="page-count"> · 本页 {{ books.length }} 册</span></span>
            <div class="page-actions">
              <button class="button secondary small" type="button" :disabled="booksLoading || page === 0"
                @click="previousPage"><Icon name="left" />上一页</button>
              <button class="button secondary small" type="button" :disabled="booksLoading || !nextPageToken"
                @click="nextPage">下一页<Icon name="right" /></button>
            </div>
          </footer>
        </template>

        <div v-else class="empty-state full-empty">
          <span class="empty-icon"><Icon name="stack" /></span>
          <h2 id="book-heading">{{ shelvesLoading ? '正在加载藏书' : shelvesError ? '暂时无法查看藏书' : '从一个书架开始' }}</h2>
          <p v-if="!shelvesLoading">{{ shelvesError ? '连接成功后，这里会显示书架中的图书。' : '创建书架后，就可以在这里添加和管理图书。' }}</p>
          <button v-if="!shelvesLoading && !shelvesError" type="button" class="button secondary"
            @click="openDialog('createShelf')"><Icon name="plus" />创建第一个书架</button>
        </div>
      </section>
    </div>
  </main>

  <dialog ref="dialog" class="entry-dialog" aria-labelledby="dialog-title"
    @cancel.prevent="closeDialog" @click.self="closeDialog">
    <form v-if="action" :key="action.mode" class="dialog-card" @submit.prevent="submitAction">
      <header class="dialog-heading">
        <div>
          <h2 id="dialog-title">{{ dialogTitle }}</h2>
          <p v-if="action.mode === 'createShelf'">设置书架主题和可容纳的图书数量。</p>
          <p v-if="action.mode === 'createBook'">添加到「{{ activeShelf?.theme }}」。</p>
        </div>
        <button class="icon-button" type="button" aria-label="关闭弹窗"
          :disabled="saving" @click="closeDialog"><Icon name="close" /></button>
      </header>

      <div v-if="action.mode === 'createShelf'" class="form-fields">
        <label>书架主题
          <input v-model.trim="form.theme" required autofocus placeholder="例如：文学与随笔" :disabled="saving" />
        </label>
        <label>容量（册）
          <input v-model="form.size" type="number" required min="1" max="5000" step="1" :disabled="saving" />
        </label>
      </div>

      <div v-if="action.mode === 'createBook'" class="form-fields">
        <label>书名
          <input v-model.trim="form.title" required autofocus placeholder="例如：山茶文具店" :disabled="saving" />
        </label>
        <label>作者 <span class="optional">选填</span>
          <input v-model.trim="form.author" placeholder="留空时由后端填入默认作者" :disabled="saving" />
        </label>
      </div>

      <p v-if="action.mode === 'deleteShelf'" class="delete-description">
        删除「{{ action.target.theme }}」会同时删除其中的所有图书。此操作无法撤销。
      </p>
      <p v-if="action.mode === 'deleteBook'" class="delete-description">
        确定从这个书架移除《{{ action.target.title }}》吗？
      </p>
      <p v-if="formError" role="alert" class="inline-error">{{ formError }}</p>

      <footer class="dialog-actions">
        <button class="button secondary" type="button" :autofocus="deleting" :disabled="saving"
          @click="closeDialog">取消</button>
        <button class="button" :class="deleting ? 'danger' : 'primary'" type="submit" :disabled="saving">
          {{ saving ? '处理中…' : deleting ? '确认删除' : dialogTitle }}
        </button>
      </footer>
    </form>
  </dialog>

  <div class="toast-area" role="status" aria-live="polite">
    <div v-if="notice" class="toast"><Icon name="check" />{{ notice }}</div>
  </div>
</template>
