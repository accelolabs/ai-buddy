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

Покрытие: `npm run test:coverage` — все TypeScript-исходники в `src/`, кроме тестов.
Минимум 80% по строкам, инструкциям, функциям и ветвлениям; снижение ниже порога
завершает проверку с ошибкой. HTML-отчёт: `coverage/index.html`.
