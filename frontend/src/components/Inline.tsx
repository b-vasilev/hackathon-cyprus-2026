// Renders `backtick` spans as code; everything else stays plain text.
export function Inline({ text }: Readonly<{ text: string }>) {
  return (
    <>
      {text.split(/(`[^`]+`)/).map((part, index) =>
        part.startsWith('`') && part.endsWith('`') && part.length > 1 ? (
          // eslint-disable-next-line react/no-array-index-key -- parts of a fixed string
          <code key={index}>{part.slice(1, -1)}</code>
        ) : (
          part
        )
      )}
    </>
  );
}
