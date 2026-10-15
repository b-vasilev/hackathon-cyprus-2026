import type { FoundationModule } from '../types';

export const http: FoundationModule = {
  id: 'http',
  title: 'HTTP and APIs',
  summary: 'Requests and responses, status codes, REST, JSON and authentication.',
  outcome:
    "Read an API request in the browser's network tab and tell what went wrong from the response.",
  minutes: 25,
  lessons: [
    {
      id: 'requests',
      title: 'Requests and responses',
      summary: 'What actually travels between the browser and the server.',
      blocks: [
        {
          type: 'p',
          text: 'Almost every web app works the same way: a client (browser, mobile app, another service) sends a request, and a server sends back a response. Both are plain text with a fixed structure.',
        },
        {
          type: 'code',
          code: 'POST /api/orders HTTP/1.1\nHost: shop.example.com\nAuthorization: Bearer eyJhbGciOi...\nContent-Type: application/json\n\n{ "productId": 42, "quantity": 2 }',
          caption: 'A request: method and path, then headers, then an empty line, then the body.',
        },
        {
          type: 'code',
          code: 'HTTP/1.1 201 Created\nContent-Type: application/json\nLocation: /api/orders/981\n\n{ "id": 981, "status": "pending" }',
          caption: 'The response: a status code, headers, and a body.',
        },
        {
          type: 'table',
          columns: ['Method', 'Used for', 'Safe to repeat?'],
          rows: [
            ['`GET`', 'Read something. Never changes data.', 'Yes'],
            ['`POST`', 'Create something, or trigger an action', 'No, you may get two orders'],
            ['`PUT`', 'Replace something completely', 'Yes, same result each time'],
            ['`PATCH`', 'Change part of something', 'Depends on the change'],
            ['`DELETE`', 'Remove something', 'Yes, it stays deleted'],
          ],
        },
        {
          type: 'p',
          text: '"Safe to repeat" has a name: idempotent. It matters when a request times out and the client retries. Retrying a `PUT` is harmless. Retrying a `POST` can charge a card twice, which is why payment APIs often ask for an idempotency key.',
        },
        {
          type: 'callout',
          tone: 'tip',
          title: 'Open the network tab',
          text: "In Chrome or Firefox, open DevTools (F12) and go to Network. Click any request to see its method, headers, body and response. It's the fastest way to see what an app is really doing.",
        },
      ],
    },
    {
      id: 'status-codes',
      title: 'Status codes',
      summary: 'The three-digit number that tells you who to blame.',
      blocks: [
        {
          type: 'p',
          text: 'The first digit tells you the category. 4xx means the client sent something wrong. 5xx means the server failed. That alone tells you where to start looking.',
        },
        {
          type: 'table',
          columns: ['Code', 'Meaning', 'Typical cause'],
          rows: [
            ['`200 OK`', 'It worked', 'Normal response with a body'],
            ['`201 Created`', 'Something new was created', 'After a successful `POST`'],
            ['`204 No Content`', 'It worked, nothing to return', 'After a `DELETE`'],
            [
              '`301` / `302`',
              'Look somewhere else',
              'Redirects; the new URL is in the `Location` header',
            ],
            [
              '`400 Bad Request`',
              'The request is malformed or invalid',
              'Missing field, broken JSON',
            ],
            ['`401 Unauthorized`', "We don't know who you are", 'No token, or it expired'],
            ['`403 Forbidden`', "We know who you are, and you can't do this", 'Missing permission'],
            ['`404 Not Found`', 'Nothing at this URL', 'Wrong path or ID'],
            ['`409 Conflict`', 'Clashes with the current state', 'Email already registered'],
            ['`422 Unprocessable Content`', 'Valid format, invalid data', 'Validation failed'],
            ['`429 Too Many Requests`', 'Slow down', 'Rate limit hit'],
            ['`500 Internal Server Error`', 'The server crashed', 'A bug, check server logs'],
            [
              '`502` / `503` / `504`',
              'A server or gateway in between failed',
              'Service down, overloaded or too slow',
            ],
          ],
        },
        {
          type: 'callout',
          tone: 'warning',
          title: '401 and 403 are not the same',
          text: '401 means "log in" (despite its name, it\'s about authentication). 403 means "you\'re logged in, but not allowed". Logging in again won\'t fix a 403.',
        },
      ],
    },
    {
      id: 'rest-json',
      title: 'REST and JSON',
      summary: 'The most common style for designing web APIs.',
      blocks: [
        {
          type: 'p',
          text: 'REST is a convention, not a library. You model things as resources with URLs, and use HTTP methods for what you do with them. The URL names the thing; the method is the verb.',
        },
        {
          type: 'table',
          columns: ['Request', 'Means'],
          rows: [
            ['`GET /orders`', 'List orders'],
            ['`GET /orders/981`', 'Get order 981'],
            ['`POST /orders`', 'Create an order'],
            ['`PATCH /orders/981`', 'Change part of order 981'],
            ['`DELETE /orders/981`', 'Delete order 981'],
            ['`GET /customers/7/orders`', 'Orders that belong to customer 7'],
          ],
        },
        {
          type: 'p',
          text: 'Signs of an API that ignores this: verbs in the path (`/getOrders`, `/deleteOrder?id=981`), or everything going through `POST`. They work, but nobody can guess the next endpoint.',
        },
        {
          type: 'p',
          text: 'Request and response bodies are usually JSON:',
        },
        {
          type: 'code',
          code: '{\n  "id": 981,\n  "status": "pending",\n  "paid": false,\n  "items": [{ "productId": 42, "quantity": 2 }],\n  "note": null\n}',
          caption:
            'Strings in double quotes, numbers, `true`/`false`, `null`, arrays and objects. No comments, no trailing commas.',
        },
        {
          type: 'p',
          text: 'Most APIs describe their endpoints in an OpenAPI (Swagger) file. If a project has one, read it before guessing.',
        },
      ],
    },
    {
      id: 'auth',
      title: 'Authentication',
      summary: 'How the server knows who is calling.',
      blocks: [
        {
          type: 'p',
          text: 'Authentication answers "who are you?". Authorization answers "what are you allowed to do?". You need the first before the second.',
        },
        {
          type: 'table',
          columns: ['Approach', 'How it works', 'Typical use'],
          rows: [
            [
              'Session cookie',
              'You log in, the server sets a cookie, and the browser sends it with every request.',
              'Classic web apps',
            ],
            [
              'Bearer token',
              'The client sends `Authorization: Bearer <token>`. Often a JWT that expires after minutes or hours.',
              'SPAs, mobile apps, APIs',
            ],
            [
              'API key',
              'A long secret string in a header, identifying an app, not a person.',
              'Server-to-server calls',
            ],
            [
              'OAuth 2.0 / OpenID Connect',
              'A login provider (Google, Microsoft, Okta) issues the tokens, so the app never sees your password.',
              '"Sign in with…" and company SSO',
            ],
          ],
        },
        {
          type: 'list',
          items: [
            "Tokens and API keys are passwords. Don't log them, don't put them in URLs, don't commit them.",
            "A JWT is encoded, not encrypted. Anyone holding one can read what's inside, so never put secrets in it.",
            'Always use HTTPS. Over plain HTTP, every header, token included, is readable by anyone on the network.',
          ],
        },
      ],
    },
  ],
  quiz: [
    {
      id: 'http-codes-match',
      kind: 'fill',
      prompt: 'Match each situation to the status code the server should send back.',
      template:
        "The order was created → {{0}}\nNo token, or it expired → {{1}}\nLogged in, but not an admin → {{2}}\nOrder 999 doesn't exist → {{3}}\nThe server threw an exception → {{4}}",
      answers: ['201', '401', '403', '404', '500'],
      distractors: ['200', '302', '429'],
      explanation:
        "201 for created, 401 when the server doesn't know who you are, 403 when it knows but says no, 404 for nothing there, 500 when it's the server's fault.",
    },
    {
      id: 'http-rest-design',
      kind: 'cards',
      look: 'file',
      prompt: 'Which request follows REST conventions for deleting order 981?',
      cards: [
        {
          title: 'GET /deleteOrder?id=981',
          flaw: 'GET',
          feedback:
            'A `GET` must never change data. Browsers, caches and link previews send `GET` requests freely, so this could delete orders by accident.',
        },
        {
          title: 'POST /orders/981/delete',
          flaw: '/delete',
          feedback: 'It works, but the verb is in the URL. HTTP already has a method for this.',
        },
        {
          title: 'DELETE /orders/981',
          correct: true,
          feedback: 'The URL names the resource and the method says what to do with it.',
        },
        {
          title: 'DELETE /orders',
          body: '{ "id": 981 }',
          flaw: 'DELETE /orders',
          feedback:
            'That reads as "delete all orders". Put the ID in the URL of the thing you\'re deleting.',
        },
      ],
    },
    {
      id: 'http-403',
      kind: 'single',
      scenario:
        'A user says "Save" does nothing. In the network tab, `PATCH /api/projects/12` returns `403 Forbidden`. Other pages work fine.',
      prompt: "What's the most likely cause?",
      options: [
        'They need to log in again',
        "They're logged in, but don't have permission to edit project 12",
        "Project 12 doesn't exist",
        'The server is down',
      ],
      correct: [1],
      explanation:
        "403 means the server knows who they are and refuses this action. Logging in again won't help; check their role or the project's permissions.",
    },
    {
      id: 'http-retry',
      kind: 'single',
      scenario: 'A request to place an order times out. The client automatically retries it.',
      prompt: 'Why is this riskier for `POST /orders` than for `PUT /orders/981`?',
      options: [
        '`POST` is slower',
        'The first `POST` may have worked, so a retry can create a second order. Repeating a `PUT` gives the same result.',
        "`PUT` requests can't time out",
        'There is no difference',
      ],
      correct: [1],
      explanation:
        "`PUT` is idempotent: sending it twice leaves the same state. `POST` isn't, which is why payment and order APIs often take an idempotency key.",
    },
    {
      id: 'http-auth-safety',
      kind: 'multiple',
      prompt: 'Which of these are good practice with tokens and API keys?',
      options: [
        'Send them in the `Authorization` header over HTTPS',
        'Keep them out of logs and commits',
        "Put them in the URL so they're easy to test, like `?token=...`",
        "Store a user's password inside a JWT",
      ],
      correct: [0, 1],
      explanation:
        'URLs end up in logs, browser history and analytics. A JWT can be decoded by anyone holding it, so it must never contain secrets.',
    },
  ],
};
