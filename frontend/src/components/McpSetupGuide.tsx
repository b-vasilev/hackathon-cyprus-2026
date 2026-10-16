export function McpSetupGuide({ url }: { readonly url: string }) {
  const escapedURL = url.replaceAll("'", String.raw`'\''`);
  const quotedURL = `'${escapedURL}'`;
  const codexCommands = `codex mcp add wayfinder --url ${quotedURL}\ncodex mcp login wayfinder`;
  const guides = [
    {
      name: 'Claude Code CLI',
      instructions: 'Run this command, then open Claude Code and use /mcp to sign in to Wayfinder.',
      code: `claude mcp add --transport http --scope user wayfinder ${quotedURL}`,
      docs: 'https://code.claude.com/docs/en/mcp',
    },
    {
      name: 'Codex CLI',
      instructions: 'Run these commands and complete browser sign-in.',
      code: codexCommands,
      docs: 'https://developers.openai.com/codex/mcp/',
    },
    {
      name: 'Claude Desktop',
      instructions: [
        'Open Customize > Connectors > + Add > Add custom connector.',
        'Name it Wayfinder and paste the server URL above.',
        'Follow the setup prompts, sign in to Wayfinder, and approve read access.',
        'In a chat, open + > Connectors and enable Wayfinder.',
      ],
      note: 'Use a public HTTPS URL; localhost will not work. For Team or Enterprise, ask your workspace owner to add the connector first.',
      docs: 'https://support.claude.com/en/articles/11175166-get-started-with-custom-connectors-using-remote-mcp',
    },
    {
      name: 'Codex desktop',
      instructions: [
        'Open a terminal on the same computer as the Codex app. The Codex CLI must be installed.',
        'Run the commands below, then sign in to Wayfinder in your browser and approve read access.',
      ],
      code: codexCommands,
      note: 'Restart the Codex app and start a new chat. The app and CLI share the MCP settings in ~/.codex/config.toml.',
      docs: 'https://developers.openai.com/codex/mcp/',
    },
    {
      name: 'Pi',
      instructions:
        'Use a current Pi version with pi mcp support. Run these commands and complete browser sign-in.',
      code: `pi mcp add wayfinder --url ${quotedURL}\npi mcp login wayfinder`,
      docs: 'https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/cli.md#mcp-server-management',
    },
    {
      name: 'OpenCode',
      instructions:
        'Merge this entry into the mcp section of opencode.json, then run opencode mcp auth wayfinder to sign in.',
      code: JSON.stringify({ mcp: { wayfinder: { type: 'remote', url } } }, null, 2),
      docs: 'https://opencode.ai/docs/mcp-servers/',
    },
    {
      name: 'Cursor',
      instructions:
        'Merge this entry into ~/.cursor/mcp.json (all projects) or .cursor/mcp.json (this project). Open MCP settings and connect Wayfinder to sign in.',
      code: JSON.stringify({ mcpServers: { wayfinder: { url } } }, null, 2),
      docs: 'https://cursor.com/docs/mcp',
    },
    {
      name: 'VS Code / GitHub Copilot',
      instructions:
        'In the Command Palette, run MCP: Add Server. Choose HTTP, paste the server URL above, and name it wayfinder. Choose where to save it, start the server, and sign in when prompted.',
      docs: 'https://code.visualstudio.com/docs/agent-customization/mcp-servers',
    },
  ];

  return (
    <section className="mcp-setup-guide" aria-labelledby="mcp-clients-title">
      <h2 id="mcp-clients-title">Setup by app</h2>
      <p>Prefer to set it up yourself? Choose your app for short steps.</p>
      {guides.map((guide) => (
        <details key={guide.name}>
          <summary>{guide.name}</summary>
          {Array.isArray(guide.instructions) ? (
            <ol>
              {guide.instructions.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ol>
          ) : (
            <p>{guide.instructions}</p>
          )}
          {guide.code !== undefined && (
            <pre>
              <code>{guide.code}</code>
            </pre>
          )}
          {guide.note !== undefined && <p className="field-help">{guide.note}</p>}
          <a href={guide.docs} target="_blank" rel="noreferrer">
            {guide.name} setup docs
          </a>
        </details>
      ))}
      <p className="field-help">
        Other apps: add the URL as a remote Streamable HTTP MCP server with OAuth browser sign-in.
        After setup, restart your AI app or CLI session and start a new chat.
      </p>
    </section>
  );
}
