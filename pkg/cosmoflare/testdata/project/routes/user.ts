import { Hono } from 'hono'
import { kvCache } from '../middleware/cache'

userRoutes.get('/me', authMiddleware, kvCache({ keyFn: (c) => `api:user:me:${c.req.query('full') || '0'}`, ttl: 300 }), async (c) => {
  const user = c.get('user')
  return c.json(user)
})
