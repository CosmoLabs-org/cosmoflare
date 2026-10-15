export const kvCache = (options: { keyFn: (c: any) => string | null, ttl?: number }) => {
  return async (c: any, next: () => Promise<void>) => {
    const key = options.keyFn(c)
    if (key) {
      const hit = await cache.get(key)
      if (hit) {
        c.header('Cache-Control', 'public, max-age=300')
        return c.json(JSON.parse(hit))
      }
    }
    await next()
    if (c.res.status === 200) {
      await cache.put(key, JSON.stringify(await c.res.json()))
    }
  }
}

export async function invalidateChurchCache(slug: string) {
  await cache.delete(churchCacheKey(slug))
  await cache.delete(`page:/church/${slug}`)
}
