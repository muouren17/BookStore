import assert from 'node:assert/strict'
import { once } from 'node:events'
import { createServer } from 'node:http'
import { test } from 'node:test'
import { createBookstoreApi } from '../src/api/bookstore.js'

async function withGateway(t, handler) {
  const server = createServer(handler)
  server.listen(0, '127.0.0.1')
  await once(server, 'listening')
  t.after(() => new Promise((resolve) => server.close(resolve)))
  return createBookstoreApi(`http://127.0.0.1:${server.address().port}`)
}

function sendJSON(res, status, body) {
  res.writeHead(status, { 'Content-Type': 'application/json' }).end(JSON.stringify(body))
}

test('翻页时完整保留游标与 64 位 ID', async (t) => {
  const api = await withGateway(t, (req, res) => {
    if (req.url !== '/v1/shelves/9007199254740993/books?page_token=a%2B%2F%3D') {
      res.writeHead(400).end()
      return
    }
    sendJSON(res, 200, {
      books: [{ id: '9007199254740995', title: '山茶文具店', author: '小川糸' }],
      nextPageToken: 'next',
    })
  })

  const page = await api.listBooks('9007199254740993', 'a+/=')
  assert.equal(page.books[0].id, '9007199254740995')
  assert.equal(page.nextPageToken, 'next')
})

test('添加图书时发送网关要求的扁平 JSON', async (t) => {
  const api = await withGateway(t, async (req, res) => {
    const chunks = []
    for await (const chunk of req) chunks.push(chunk)
    const body = JSON.parse(Buffer.concat(chunks).toString())
    if (req.method !== 'POST' || req.url !== '/v1/shelves/7/books' ||
        body.title !== '小王子' || body.author !== '圣埃克苏佩里' || 'book' in body) {
      res.writeHead(400).end()
      return
    }
    sendJSON(res, 200, { book: { id: '12', ...body } })
  })

  assert.equal((await api.createBook('7', { title: '小王子', author: '圣埃克苏佩里' })).id, '12')
})

test('将空列表响应转为空数组', async (t) => {
  const api = await withGateway(t, (_req, res) => sendJSON(res, 200, {}))
  assert.deepEqual(await api.listShelves(), [])
  assert.deepEqual(await api.listBooks('7'), { books: [], nextPageToken: '' })
})

test('读取网关错误信息，并处理非 JSON 错误', async (t) => {
  const api = await withGateway(t, (req, res) => {
    if (req.url === '/v1/shelves/7/books') {
      sendJSON(res, 400, { code: 3, message: '书架容量已满' })
      return
    }
    res.writeHead(503, { 'Content-Type': 'text/plain' }).end('Service unavailable')
  })
  await assert.rejects(api.createBook('7', { title: '小王子', author: '' }), /书架容量已满/)
  await assert.rejects(api.listShelves(), /503/)
})

test('接受删除操作的空响应', async (t) => {
  const api = await withGateway(t, (req, res) => {
    const deleted = req.method === 'DELETE' && req.url === '/v1/shelves/7/books/12'
    res.writeHead(deleted ? 204 : 400).end()
  })
  await api.deleteBook('7', '12')
})
