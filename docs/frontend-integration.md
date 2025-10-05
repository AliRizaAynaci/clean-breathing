# Frontend Integration Guide

This backend issues an HTTP-only session cookie (`session_token`) after the Google OAuth callback succeeds. The frontend never sees the JWT directly; instead, it must let the browser store the cookie for the backend domain and include it on every API request.

## OAuth flow

1. Redirect the user to `GET /auth/google/login`.
2. Google redirects back to `GET /auth/google/callback` with the authorization code.
3. The backend exchanges the code, creates the user, and sets the `session_token` cookie (scoped to the API domain, `SameSite=None`, `Secure=true`).
4. The backend redirects the browser to the URL defined in `FRONTEND_REDIRECT_URL` or `{FRONTEND_URI}/dashboard`.

Because the cookie is `HttpOnly`, the frontend cannot read it with JavaScript. Authentication relies on the browser automatically attaching the cookie when calling the API.

## Making authenticated requests

Whether you use `fetch`, `axios`, or another HTTP client, make sure that every request to this backend is sent **with credentials**:

```tsx
// fetch example
await fetch(process.env.NEXT_PUBLIC_API_URL + "/me", {
  method: "GET",
  credentials: "include", // critical!
});
```

```ts
// axios example
const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL,
  withCredentials: true,
});

const profile = await api.get("/me");
```

In Next.js, you can set `axios.defaults.withCredentials = true` or pass `credentials: 'include'` in every `fetch`. When using the App Router, server actions must also forward cookies via `headers: { Cookie: cookies().toString() }`.

## CORS and cookies checklist

- `FRONTEND_URI` and `FRONTEND_EXTRA_ORIGINS` should contain every origin that will call the API (for example, your Vercel Preview and Production URLs).
- The frontend must call the API using HTTPS and `credentials: 'include'`.
- Configure `SESSION_COOKIE_SECURE=true` and `SESSION_COOKIE_SAMESITE=None` in production.
- Only set `SESSION_COOKIE_DOMAIN` if the backend should share a cookie across multiple subdomains. Leave it empty when the API lives on a unique host such as `*.herokuapp.com`.

## Local development

When running everything on `localhost`, you can set `SESSION_COOKIE_SECURE=false` so that browsers allow the cookie over HTTP. Keep `SESSION_COOKIE_SAMESITE=Lax` or `None` depending on your frontend port.

Following this checklist ensures the cookie is stored on the backend domain and automatically included in subsequent requests, allowing `/me` and other protected endpoints to recognize the user.
