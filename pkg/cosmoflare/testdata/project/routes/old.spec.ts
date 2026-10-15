import { kvCache } from '../middleware/cache'

oldRoutes.get('/gone', kvCache({ keyFn: () => 'api:old:gone', ttl: 60 }), async (c) => {
  return c.json({ gone: true })
})
