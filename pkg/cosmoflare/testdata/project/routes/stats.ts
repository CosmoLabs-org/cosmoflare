import { Hono } from 'hono'

statsRoutes.get('/summary', async (c) => {
  const rows = await env.DB.prepare(
    `SELECT country, COUNT(*) FROM churches GROUP BY country`
  ).all()
  return c.json(rows.results)
})

statsRoutes.get('/detail', async (c) => {
  const where = buildWhere(c)
  const rows = await env.DB.prepare(
    `SELECT country, COUNT(*) FROM churches ${where} GROUP BY country`
  ).all()
  return c.json(rows.results)
})
