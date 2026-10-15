import { Hono } from 'hono'
import { kvCache } from '../middleware/cache'

metaRoutes.get('/stats', kvCache({ keyFn: () => 'api:meta:stats', ttl: 600 }), async (c) => {
  const rows = await env.DB.prepare('SELECT name FROM countries').all()
  return c.json(rows.results)
})

metaRoutes.get('/cities', kvCache({ keyFn: (c) => `api:meta:cities:${c.req.query('country_code') || 'all'}`, ttl: 21600 }), async (c) => {
  const rows = await env.DB.prepare('SELECT name FROM cities').all()
  return c.json(rows.results)
})

metaRoutes.get('/states', kvCache({ keyFn: (c) => (c.req.query('country_code') ? `api:meta:states:${c.req.query('country_code')}` : null), ttl: 21600 }), async (c) => {
  const rows = await env.DB.prepare('SELECT name FROM states').all()
  return c.json(rows.results)
})
