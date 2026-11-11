const shelvesPath = '/v1/shelves'

const errorMessages = {
  'shelf cant contain book out of capacity': '这个书架已满，请移除一本图书后再试。',
  'find no target shelf': '找不到这个书架，请刷新后重试。',
  'find no target book': '找不到这本图书，请刷新后重试。',
  'theme cant be nil': '请输入书架主题。',
}

export function createBookstoreApi(baseUrl = '') {
  async function request(path, { method = 'GET', body, signal } = {}) {
    let response
    try {
      response = await fetch(baseUrl + path, {
        method,
        headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
        body: body === undefined ? undefined : JSON.stringify(body),
        signal,
      })
    } catch (error) {
      if (signal?.aborted) throw error
      throw new Error('连接失败，请确认 BookStore 后端已启动。', { cause: error })
    }

    const text = await response.text()
    let data = {}
    if (text) {
      try {
        data = JSON.parse(text) ?? {}
      } catch {
        if (response.ok) throw new Error('服务返回了无法读取的响应，请稍后重试。')
      }
    }

    if (!response.ok) {
      const message = typeof data.message === 'string' ? data.message : ''
      throw new Error(errorMessages[message] || message || '请求失败（HTTP ' + response.status + '）。')
    }

    return data
  }

  const shelfPath = (id) => shelvesPath + '/' + encodeURIComponent(id)
  const booksPath = (id) => shelfPath(id) + '/books'

  return {
    async listShelves(signal) {
      const data = await request(shelvesPath, { signal })
      return data.shelf ?? []
    },
    async createShelf({ theme, size }) {
      const data = await request(shelvesPath, { method: 'POST', body: { theme, size } })
      return data.shelf
    },
    deleteShelf(id) {
      return request(shelfPath(id), { method: 'DELETE' })
    },
    async listBooks(shelfId, pageToken = '', signal) {
      const query = pageToken ? '?' + new URLSearchParams({ page_token: pageToken }) : ''
      const data = await request(booksPath(shelfId) + query, { signal })
      return { books: data.books ?? [], nextPageToken: data.nextPageToken ?? '' }
    },
    async createBook(shelfId, { title, author }) {
      const data = await request(booksPath(shelfId), { method: 'POST', body: { title, author } })
      return data.book
    },
    deleteBook(shelfId, bookId) {
      return request(booksPath(shelfId) + '/' + encodeURIComponent(bookId), { method: 'DELETE' })
    },
  }
}
