# Фронтенд AI Buddy

React, TypeScript, Vite, TanStack Query и Zustand. Запуск и переменные описаны
в [корневом README](../README.md).

```sh
npm ci
npm run dev
```

Проверки: `npm run build`, `npm run lint:check`, `npm run format:check`, `npm test`.
API вызывается через `/api` с cookie `demo_user`; Vite проксирует запросы на порт 8080.
В Docker статика и API доступны через Nginx. История чата не сохраняется в браузере.
