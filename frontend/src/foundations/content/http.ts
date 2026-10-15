import type { FoundationModule } from '../types';

const BEARER = 'Bearer token';
const GET_981 = '`GET /orders/981`';
const PUT_981 = '`PUT /orders/981`';
const POST_ORDERS = '`POST /orders`';
const AGAIN = ' again';
const OK_200 = '`200 OK`';
const CHECKS_OK = 'login and permission OK';
const NOT_REACHED = 'not reached';

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
      summary: 'What a request and a response contain.',
      blocks: [
        {
          type: 'p',
          text: 'Almost every web app works in the same way. A client sends a request, and a server sends back a response. The client is the program that asks: a browser, a mobile app or another service. The request and the response are both plain text with a fixed structure.',
        },
        {
          type: 'anatomy',
          title: 'A request',
          segments: [
            { text: 'POST', part: 'method' },
            { text: ' ' },
            { text: '/api/orders', part: 'path' },
            { text: ' HTTP/1.1\n' },
            { text: 'Host: shop.example.com', part: 'headers' },
            { text: '\n' },
            { text: 'Authorization: Bearer eyJhbGciOi...', part: 'auth' },
            { text: '\n' },
            { text: 'Content-Type: application/json', part: 'headers' },
            { text: '\n' },
            { text: '\n', part: 'empty' },
            { text: '{ "productId": 42, "quantity": 2 }', part: 'body' },
          ],
          parts: [
            {
              id: 'method',
              label: 'Method',
              explanation:
                'What you want to do. `POST` means "create something" or "start an action".',
            },
            {
              id: 'path',
              label: 'Path',
              explanation:
                'Which thing on the server the request is about. Here it is the list of orders.',
            },
            {
              id: 'headers',
              label: 'Headers',
              explanation:
                'Extra information about the request, one `Name: value` per line. `Host` names the server. `Content-Type` says that the body is JSON.',
            },
            {
              id: 'auth',
              label: 'Authorization',
              explanation:
                'A header that says who is calling. Here it carries a token. The lesson on authentication explains this.',
            },
            {
              id: 'empty',
              label: 'Empty line',
              explanation: 'An empty line ends the headers. Everything after it is the body.',
            },
            {
              id: 'body',
              label: 'Body',
              explanation:
                'The data you send. Here it is JSON: the product and how many you want. A `GET` request usually has no body.',
            },
          ],
        },
        {
          type: 'code',
          code: 'HTTP/1.1 201 Created\nContent-Type: application/json\nLocation: /api/orders/981\n\n{ "id": 981, "status": "pending" }',
          caption: 'The response. It has a status code, headers and a body.',
        },
        {
          type: 'p',
          text: 'Every request has a method. The method says what you want to do with the thing at that path.',
        },
        {
          type: 'table',
          columns: ['Method', 'Used for', 'Safe to repeat?'],
          rows: [
            ['`GET`', 'Read something. It never changes data.', 'Yes'],
            ['`POST`', 'Create something, or start an action', 'No, you may get two orders'],
            ['`PUT`', 'Replace something completely', 'Yes, same result each time'],
            ['`PATCH`', 'Change part of something', 'It depends on the change'],
            ['`DELETE`', 'Remove something', 'Yes, it stays deleted'],
          ],
        },
        {
          type: 'p',
          text: '"Safe to repeat" has a name: idempotent. A request is idempotent when sending it twice gives the same result as sending it once. This matters when a request times out and the client sends it again. This is called a retry. Retrying a `PUT` does no harm. Retrying a `POST` can charge a card twice. For this reason, payment APIs often ask for an idempotency key.',
        },
        {
          type: 'sequence',
          title: 'What happens when you send it twice',
          actors: ['Client', 'Server', 'Database'],
          scenes: [
            {
              name: 'GET',
              summary: 'A `GET` only reads, so two requests give the same result.',
              description: 'The data does not change at all.',
              messages: [
                { from: 0, to: 1, label: GET_981 },
                { from: 2, to: 2, label: 'reads order 981' },
                { from: 1, to: 0, label: OK_200, tone: 'brand' },
                { from: 0, to: 1, label: GET_981 + AGAIN },
                { from: 2, to: 2, label: 'order 981 unchanged' },
                { from: 1, to: 0, label: '`200 OK`, same order', tone: 'brand' },
              ],
            },
            {
              name: 'POST',
              summary: 'Each `POST` creates something new, so a retry makes a second order.',
              description: 'This is why payment APIs often ask for an idempotency key.',
              messages: [
                { from: 0, to: 1, label: POST_ORDERS },
                { from: 2, to: 2, label: 'orders: 1' },
                { from: 1, to: 0, label: '`201 Created`, order 981', tone: 'brand' },
                { from: 0, to: 1, label: POST_ORDERS + AGAIN },
                { from: 2, to: 2, label: 'orders: 2', tone: 'danger' },
                { from: 1, to: 0, label: '`201 Created`, order 982', tone: 'warning' },
              ],
            },
            {
              name: 'PUT',
              summary: 'A `PUT` replaces the order with the same data, so a retry changes nothing.',
              messages: [
                { from: 0, to: 1, label: PUT_981 },
                { from: 2, to: 2, label: 'order 981 replaced' },
                { from: 1, to: 0, label: OK_200, tone: 'brand' },
                { from: 0, to: 1, label: PUT_981 + AGAIN },
                { from: 2, to: 2, label: 'order 981 unchanged' },
                { from: 1, to: 0, label: OK_200, tone: 'brand' },
              ],
            },
            {
              name: 'DELETE',
              summary: 'After the first `DELETE`, the order stays deleted.',
              description:
                'The second response can be different, often `404`. The data ends up the same.',
              messages: [
                { from: 0, to: 1, label: '`DELETE /orders/981`' },
                { from: 2, to: 2, label: 'order 981 deleted' },
                { from: 1, to: 0, label: '`204 No Content`', tone: 'brand' },
                { from: 0, to: 1, label: '`DELETE /orders/981` again' },
                { from: 2, to: 2, label: 'still deleted' },
                { from: 1, to: 0, label: '`204` or `404`', tone: 'brand' },
              ],
            },
          ],
        },
        {
          type: 'callout',
          tone: 'tip',
          title: 'Open the network tab',
          text: 'In Chrome or Firefox, press F12 to open DevTools, the developer tools of the browser. Go to the Network tab. Click any request to see its method, headers, body and response. This is the fastest way to see what an app really does.',
        },
      ],
    },
    {
      id: 'status-codes',
      title: 'Status codes',
      summary: 'How the status code tells you where a problem is.',
      blocks: [
        {
          type: 'p',
          text: 'Every response has a status code, a three-digit number. The first digit tells you the category. 4xx means the client sent something wrong. 5xx means the server failed. This tells you where to start looking.',
        },
        {
          type: 'p',
          text: 'A request passes through several checks on the server. The status code often tells you which check stopped it.',
        },
        {
          type: 'sequence',
          title: 'Where the request stops',
          actors: ['Browser', 'Server', 'Database'],
          scenes: [
            {
              name: '200 OK',
              summary: 'The request passes every check and the server sends the data back.',
              messages: [
                { from: 0, to: 1, label: GET_981 },
                { from: 1, to: 1, label: CHECKS_OK, tone: 'brand' },
                { from: 1, to: 2, label: 'find order 981' },
                { from: 2, to: 1, label: 'order 981' },
                { from: 1, to: 0, label: OK_200, tone: 'brand' },
              ],
            },
            {
              name: '401',
              summary: 'The server does not know who you are, so it stops at the login check.',
              description: 'Logging in again usually fixes this.',
              messages: [
                { from: 0, to: 1, label: GET_981 },
                { from: 1, to: 1, label: 'checks login: no token', tone: 'danger' },
                { from: 2, to: 2, label: NOT_REACHED, tone: 'muted' },
                { from: 1, to: 0, label: '`401 Unauthorized`', tone: 'danger' },
              ],
            },
            {
              name: '403',
              summary: 'The server knows who you are, but you are not allowed to do this.',
              description:
                'Logging in again does not help. Check the role or permissions of the user.',
              messages: [
                { from: 0, to: 1, label: GET_981 },
                { from: 1, to: 1, label: 'checks login: user is known', tone: 'brand' },
                { from: 1, to: 1, label: 'checks permission: missing', tone: 'danger' },
                { from: 2, to: 2, label: NOT_REACHED, tone: 'muted' },
                { from: 1, to: 0, label: '`403 Forbidden`', tone: 'danger' },
              ],
            },
            {
              name: '404',
              summary: 'All checks pass, but there is nothing at this URL.',
              description: 'The path or the ID is wrong. Here, order 999 does not exist.',
              messages: [
                { from: 0, to: 1, label: '`GET /orders/999`' },
                { from: 1, to: 1, label: CHECKS_OK, tone: 'brand' },
                { from: 1, to: 2, label: 'find order 999' },
                { from: 2, to: 1, label: 'nothing found', tone: 'warning' },
                { from: 1, to: 0, label: '`404 Not Found`', tone: 'danger' },
              ],
            },
            {
              name: '500',
              summary: 'The request was fine, but the server crashed while it worked on it.',
              description: 'This is a bug on the server. Check the server logs.',
              messages: [
                { from: 0, to: 1, label: GET_981 },
                { from: 1, to: 1, label: CHECKS_OK, tone: 'brand' },
                { from: 1, to: 1, label: 'server code throws an error', tone: 'danger' },
                { from: 1, to: 0, label: '`500 Internal Server Error`', tone: 'danger' },
              ],
            },
          ],
        },
        {
          type: 'table',
          columns: ['Code', 'Meaning', 'Typical cause'],
          rows: [
            ['`200 OK`', 'It worked', 'Normal response with a body'],
            ['`201 Created`', 'Something new was created', 'After a successful `POST`'],
            ['`204 No Content`', 'It worked, and there is nothing to return', 'After a `DELETE`'],
            [
              '`301` / `302`',
              'Look somewhere else',
              'A redirect. The new URL is in the `Location` header.',
            ],
            [
              '`400 Bad Request`',
              'The request is broken or not valid',
              'A missing field, or broken JSON',
            ],
            [
              '`401 Unauthorized`',
              'The server does not know who you are',
              'No token, or it expired',
            ],
            [
              '`403 Forbidden`',
              'The server knows who you are, and you are not allowed to do this',
              'Missing permission',
            ],
            ['`404 Not Found`', 'There is nothing at this URL', 'Wrong path or ID'],
            [
              '`409 Conflict`',
              'The request does not fit the current state',
              'The email is already registered',
            ],
            [
              '`422 Unprocessable Content`',
              'The format is valid, but the data is not',
              'Validation failed',
            ],
            ['`429 Too Many Requests`', 'Send fewer requests', 'You went over the rate limit'],
            ['`500 Internal Server Error`', 'The server crashed', 'A bug. Check the server logs.'],
            [
              '`502` / `503` / `504`',
              'A server or gateway between you and the app failed',
              'A service is down, overloaded or too slow',
            ],
          ],
        },
        {
          type: 'p',
          text: 'A rate limit is the highest number of requests the server accepts in a period of time. A gateway is a server that passes requests on to other servers.',
        },
        {
          type: 'callout',
          tone: 'warning',
          title: '401 and 403 are different',
          text: '401 means "log in". Its name says "Unauthorized", but it is about authentication: who you are. 403 means "you are logged in, but you are not allowed". Logging in again does not fix a 403.',
        },
      ],
    },
    {
      id: 'rest-json',
      title: 'REST and JSON',
      summary: 'The most common way to design web APIs.',
      blocks: [
        {
          type: 'p',
          text: 'REST is a set of rules for designing an API. It is a convention, and you do not install it as a library. You treat each thing, such as an order or a customer, as a resource with its own URL. The URL names the thing. The HTTP method says what to do with it.',
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
          text: 'Some APIs do not follow these rules. They put verbs in the path, like `/getOrders` or `/deleteOrder?id=981`. Or they send everything with `POST`. These APIs work, but nobody can guess the next endpoint. An endpoint is one method and URL that the API accepts.',
        },
        {
          type: 'p',
          text: 'Request and response bodies are usually JSON, a simple text format for data:',
        },
        {
          type: 'code',
          code: '{\n  "id": 981,\n  "status": "pending",\n  "paid": false,\n  "items": [{ "productId": 42, "quantity": 2 }],\n  "note": null\n}',
          caption:
            'JSON has strings in double quotes, numbers, `true`/`false`, `null`, arrays and objects. It has no comments and no comma after the last item.',
        },
        {
          type: 'p',
          text: 'Most APIs describe their endpoints in an OpenAPI file, also called Swagger. If a project has one, read it before you guess.',
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
          text: 'Authentication answers the question "who are you?". Authorization answers the question "what are you allowed to do?". The server needs the first answer before it can check the second.',
        },
        {
          type: 'table',
          columns: ['Approach', 'How it works', 'Typical use'],
          rows: [
            [
              'Session cookie',
              'You log in, and the server sets a cookie. The browser sends this cookie with every request.',
              'Classic web apps',
            ],
            [
              BEARER,
              'The client sends `Authorization: Bearer <token>`. This is often a JWT that expires after minutes or hours.',
              'SPAs (single-page apps), mobile apps, APIs',
            ],
            [
              'API key',
              'A long secret string in a header. It identifies an app, and not a person.',
              'Calls from one server to another',
            ],
            [
              'OAuth 2.0 / OpenID Connect',
              'A login provider (Google, Microsoft, Okta) gives out the tokens. The app never sees your password.',
              '"Sign in with…" and company SSO (single sign-on)',
            ],
          ],
        },
        {
          type: 'p',
          text: 'A cookie is a small piece of data that the browser stores and sends back to the server. A JWT (JSON Web Token) is a token that holds data about the user in an encoded form.',
        },
        {
          type: 'sequence',
          title: 'What the client sends each time',
          actors: ['Client', 'Server'],
          scenes: [
            {
              name: 'Session cookie',
              summary: 'You log in once, and the browser then sends the cookie with every request.',
              description: 'Here the client is a browser. It adds the cookie by itself.',
              messages: [
                { from: 0, to: 1, label: 'log in: name + password' },
                { from: 1, to: 1, label: 'creates a session' },
                { from: 1, to: 0, label: '`Set-Cookie: session=…`', tone: 'soft' },
                { from: 0, to: 1, label: 'next request + `Cookie`', tone: 'brand' },
                { from: 1, to: 1, label: 'finds your session', tone: 'brand' },
              ],
            },
            {
              name: BEARER,
              summary: 'The client gets a token and puts it in the `Authorization` header.',
              description:
                'The token is often a JWT. It expires after minutes or hours, and then the server answers `401`.',
              messages: [
                { from: 0, to: 1, label: 'log in' },
                { from: 1, to: 0, label: 'token, often a JWT', tone: 'soft' },
                { from: 0, to: 1, label: '`Authorization: Bearer …`', tone: 'brand' },
                { from: 1, to: 1, label: 'checks the token', tone: 'brand' },
                { from: 1, to: 0, label: OK_200 },
              ],
            },
            {
              name: 'API key',
              summary: 'A server sends the same secret key in a header with every request.',
              description:
                'There is no login step. The key identifies an app, and not a person. The header name depends on the API.',
              messages: [
                { from: 0, to: 1, label: 'request + `X-API-Key: …`', tone: 'brand' },
                { from: 1, to: 1, label: 'checks the key: knows the app', tone: 'brand' },
                { from: 1, to: 0, label: OK_200 },
                { from: 0, to: 1, label: 'next request + same key', tone: 'brand' },
              ],
            },
          ],
        },
        {
          type: 'list',
          items: [
            'Treat tokens and API keys like passwords. Do not log them, do not put them in URLs and do not commit them.',
            'A JWT is encoded, but it is not encrypted. Anyone who has the token can read what is inside it. Never put secrets in a JWT.',
            'Always use HTTPS. With plain HTTP, anyone on the network can read every header, and this includes the token.',
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
        "The order was created → {{0}}\nNo token, or it expired → {{1}}\nLogged in, but not an admin → {{2}}\nOrder 999 doesn't exist → {{3}}\nThe server threw an error → {{4}}",
      answers: ['201', '401', '403', '404', '500'],
      distractors: ['200', '302', '429'],
      explanation:
        '201 means something was created. 401 means the server does not know who you are. 403 means it knows you and says no. 404 means there is nothing at the URL. 500 means the server failed.',
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
            'A `GET` must never change data. Browsers, caches and link previews send `GET` requests freely. This request could delete orders by accident.',
        },
        {
          title: 'POST /orders/981/delete',
          flaw: '/delete',
          feedback: 'It works, but the verb is in the URL. HTTP already has a method for this.',
        },
        {
          title: 'DELETE /orders/981',
          correct: true,
          feedback: 'The URL names the resource, and the method says what to do with it.',
        },
        {
          title: 'DELETE /orders',
          body: '{ "id": 981 }',
          flaw: 'DELETE /orders',
          feedback:
            'This reads as "delete all orders". Put the ID in the URL of the thing you want to delete.',
        },
      ],
    },
    {
      id: 'http-403',
      kind: 'single',
      scenario:
        'A user says that "Save" does nothing. In the network tab, `PATCH /api/projects/12` returns `403 Forbidden`. Other pages work.',
      prompt: 'What is the most likely cause?',
      options: [
        'They need to log in again',
        'They are logged in, but they do not have permission to edit project 12',
        'Project 12 does not exist',
        'The server is down',
      ],
      correct: [1],
      explanation:
        '403 means the server knows who they are and refuses this action. Logging in again does not help. Check their role or the permissions of the project.',
    },
    {
      id: 'http-retry',
      kind: 'single',
      scenario: 'A request to place an order times out. The client sends it again automatically.',
      prompt: 'Why is this more dangerous for `POST /orders` than for `PUT /orders/981`?',
      options: [
        '`POST` is slower',
        'The first `POST` may have worked, so a retry can create a second order. Repeating a `PUT` gives the same result.',
        '`PUT` requests cannot time out',
        'There is no difference',
      ],
      correct: [1],
      explanation:
        '`PUT` is idempotent: if you send it twice, the data ends up the same. `POST` is not idempotent. This is why payment and order APIs often take an idempotency key.',
    },
    {
      id: 'http-auth-safety',
      kind: 'multiple',
      prompt: 'Which of these are good practice with tokens and API keys?',
      options: [
        'Send them in the `Authorization` header over HTTPS',
        'Keep them out of logs and commits',
        'Put them in the URL so they are easy to test, like `?token=...`',
        "Store a user's password inside a JWT",
      ],
      correct: [0, 1],
      explanation:
        'URLs are saved in logs, browser history and analytics. Anyone who has a JWT can decode it, so it must never contain secrets.',
    },
  ],
};
