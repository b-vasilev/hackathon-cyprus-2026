import assert from 'node:assert/strict';
import test from 'node:test';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderToStaticMarkup } from 'react-dom/server';
import { documentChapters, legacySections } from '../lib/passportDocument';
import { DiagramNode } from './DiagramNode';
import { DocumentContent } from './DocumentContent';
import { PartialResearchNotice } from './PartialResearchNotice';
import { ObjectDetail, SectionContent } from './PassportContent';
import { PassportReview } from './PassportReview';
import { SafeMarkdown } from './SafeMarkdown';
import type { Chapter, ProductObject, Revision, Section } from '../lib/api';

const object = (id: string): ProductObject => ({
  id,
  kind: 'component',
  name: id,
  summary: `${id} responsibility`,
  technology: 'Example runtime',
  status: 'implemented',
  inputs: [],
  outputs: [],
  contracts: [],
  claim_ids: [],
});

function revision(): Revision {
  return {
    id: 'revision',
    product_id: 'product',
    run_id: 'run',
    version: 1,
    schema_version: '2',
    created_at: '2026-10-15T00:00:00Z',
    published_by: '',
    published_at: '',
    published: false,
    unpublished_changes: false,
    snapshots: [],
    content: {
      model: {
        title: 'Example product',
        tagline: '',
        summary: '',
        objects: [object('First component'), object('Second component')],
        claims: [],
        relations: [],
        evidence: [],
        open_questions: [],
        scenarios: [
          {
            id: 'main-flow',
            title: 'Main flow',
            summary: 'Verified flow outcome',
            inputs: [],
            outputs: [],
            failure_modes: [],
            steps: [],
          },
        ],
      },
      views: { sections: [], diagrams: [], card_theme: null },
    },
  };
}

const section: Section = {
  id: 'overview',
  title: 'Saved overview',
  status: 'available',
  summary: 'Saved narrative',
  object_ids: [],
  claim_ids: [],
  diagram_ids: [],
  scenario_ids: [],
  checklist: [],
};

function renderNode(item: ProductObject): string {
  return renderToStaticMarkup(
    <DiagramNode
      id={item.id}
      type="architecture"
      data={{ object: item, ports: [], onInspect: () => {}, focused: false }}
      dragging={false}
      zIndex={0}
      selectable={false}
      deletable={false}
      selected={false}
      draggable={false}
      isConnectable={false}
      positionAbsoluteX={0}
      positionAbsoluteY={0}
    />
  );
}

test('published passports can publish a newly ready quiz without text changes', () => {
  for (const ready of [false, true]) {
    const client = new QueryClient({
      defaultOptions: { queries: { staleTime: Number.POSITIVE_INFINITY } },
    });
    client.setQueryData(['review', 'revision'], { flagged_at: '', edited_at: '', activity: [] });
    client.setQueryData(['learning-pack', 'revision'], {
      status: ready ? 'ready' : 'generating',
      content: ready ? { quiz: [] } : null,
      published: false,
      unpublished_changes: false,
    });
    const html = renderToStaticMarkup(
      <QueryClientProvider client={client}>
        <PassportReview
          revision={{ ...revision(), published: true }}
          chapters={[]}
          currentChapterId=""
        />
      </QueryClientProvider>
    );
    assert.equal(html.includes('Publish changes'), ready);
    assert.equal(html.includes('also publishes its quiz and first task'), ready);
    assert.equal(html.includes('The quiz is still generating'), !ready);
    client.clear();
  }
});

test('architectural roles label nodes and inspectors with a legacy kind fallback', () => {
  for (const role of ['software_system', 'runtime_unit', 'component', '']) {
    const item = { ...object('Example component'), architecture_role: role };
    const label = role === '' ? 'component' : role.replaceAll('_', ' ');
    assert.ok(renderNode(item).includes(`class="node-kind">${label}</span>`));
    assert.match(renderNode(item), /class="architecture-status"/);
    const detail = renderToStaticMarkup(
      <ObjectDetail
        object={item}
        model={revision().content.model}
        onClose={() => {}}
        onEvidence={() => {}}
      />
    );
    assert.ok(detail.includes(`>${label}</span>`));
  }
  assert.match(renderNode(object('Legacy component')), /class="node-kind">component<\/span>/);
});

test('actor and owner nodes omit lifecycle badges while inspectors retain source status', () => {
  for (const kind of ['actor', 'owner']) {
    const item = { ...object('Responsible person'), kind, status: 'unknown' };
    const html = renderNode(item);
    assert.doesNotMatch(html, /class="architecture-status"/);
    assert.ok(html.includes(`class="node-kind">${kind}</span>`));
    const detail = renderToStaticMarkup(
      <ObjectDetail
        object={item}
        model={revision().content.model}
        onClose={() => {}}
        onEvidence={() => {}}
      />
    );
    assert.match(detail, />unknown<\/span>/);
  }
});

test('Markdown renders prose while dropping HTML, media, and unsafe links', () => {
  const html = renderToStaticMarkup(
    <SafeMarkdown>{`## A useful heading

**Strong text** and \`code\`.

<script>alert('bad')</script>

<iframe src="https://untrusted.example"></iframe>

[Unsafe](javascript:alert%281%29) [Data](data:text/html,bad) [Relative](/settings) [Network](//untrusted.example)

[Source](https://example.com/source) [Email](mailto:reader@example.com)

![Description](https://untrusted.example/tracker.png)

{globalThis.alert('not executable')}`}</SafeMarkdown>
  );
  assert.match(html, /<h3>A useful heading<\/h3>/);
  assert.match(html, /<strong>Strong text<\/strong>/);
  assert.match(html, /<code>code<\/code>/);
  assert.match(html, /href="https:\/\/example.com\/source"/);
  assert.match(html, /rel="noopener noreferrer"/);
  assert.match(html, /href="mailto:reader@example.com"/);
  assert.doesNotMatch(html, /<script|<iframe|<img|javascript:|data:text|href="\/|tracker.png/);
  assert.match(html, /not executable/);
});

test('Markdown variant matrices render as safe, styled GFM tables', () => {
  const html = renderToStaticMarkup(
    <SafeMarkdown>{`| Variant | Result | Source |
| --- | --- | --- |
| Local | **In process** | [Verified source](https://example.com/local) |
| Remote | <img src="https://untrusted.example/image" onerror="alert(1)"> | [Blocked](javascript:alert%281%29) [Relative](/settings) |
| Other | <script>alert(1)</script> | [Data](data:text/html,bad) |`}</SafeMarkdown>
  );
  assert.match(html, /class="passport-table-scroll"/);
  assert.match(html, /<table class="passport-table">/);
  assert.match(html, /<thead><tr><th>Variant<\/th>/);
  assert.match(html, /<tbody><tr><td>Local<\/td>/);
  assert.match(html, /<strong>In process<\/strong>/);
  assert.match(html, /href="https:\/\/example.com\/local"/);
  assert.match(html, /rel="noopener noreferrer"/);
  assert.doesNotMatch(html, /<script|<img|onerror|javascript:|data:text|href="\//);
});

test('generic object tables retain non-technology objects and responsibility labels', () => {
  const saved = revision();
  saved.content.model.objects = [
    { ...object('Report artifact'), kind: 'entity', technology: '' },
    { ...object('Operator'), kind: 'actor', technology: '' },
  ];
  const chapter: Chapter = {
    id: 'overview',
    title: 'Responsibilities',
    question: '',
    blocks: [
      { kind: 'object_table', table_style: 'objects', object_ids: ['Operator', 'Report artifact'] },
    ],
  };
  const html = renderToStaticMarkup(
    <DocumentContent chapter={chapter} revision={saved} onObject={() => {}} onEvidence={() => {}} />
  );
  assert.match(html, /<caption>Objects and their responsibilities<\/caption>/);
  assert.match(html, /<th scope="col">Object<\/th>/);
  assert.match(html, /<th scope="col">Type<\/th>/);
  assert.match(html, /Operator responsibility/);
  assert.match(html, /Report artifact responsibility/);
  assert.match(html, /<td>actor<\/td>/);
  assert.match(html, /<td>entity<\/td>/);
  assert.ok(html.indexOf('Operator') < html.indexOf('Report artifact'));
  assert.doesNotMatch(
    html,
    /Technologies and their responsibilities|Not specified|<th scope="col">Technology/
  );
});

test('v2 follows authored chapter and block order and resolves canonical flow and object IDs', () => {
  const saved = revision();
  const chapter: Chapter = {
    id: 'architecture',
    title: 'How the product works',
    question: '',
    blocks: [
      { kind: 'markdown', markdown: 'Before the flow.' },
      { kind: 'scenario_ref', reference_id: 'main-flow' },
      { kind: 'markdown', markdown: 'After the flow, inspect the components.' },
      {
        kind: 'object_table',
        table_style: 'objects',
        object_ids: ['Second component', 'First component'],
      },
    ],
  };
  saved.content.views.document = {
    version: 2,
    chapters: [
      chapter,
      { ...chapter, id: 'overview' },
      { id: 'operations', title: 'Empty', question: '', blocks: [] },
    ],
  };
  assert.deepEqual(
    documentChapters(saved.content.views)?.map((item) => item.id),
    ['architecture', 'overview']
  );
  const html = renderToStaticMarkup(
    <DocumentContent chapter={chapter} revision={saved} onObject={() => {}} onEvidence={() => {}} />
  );
  assert.ok(html.indexOf('Before the flow.') < html.indexOf('Verified flow outcome'));
  assert.ok(html.indexOf('Verified flow outcome') < html.indexOf('After the flow'));
  assert.ok(html.indexOf('After the flow') < html.indexOf('<table'));
  assert.ok(html.indexOf('Second component') < html.indexOf('First component'));
});

test('missing references show a clear message instead of inventing content', () => {
  const chapter: Chapter = {
    id: 'architecture',
    title: 'Structure',
    question: '',
    blocks: [
      { kind: 'diagram_ref', reference_id: 'missing' },
      { kind: 'scenario_ref', reference_id: 'missing' },
    ],
  };
  const html = renderToStaticMarkup(
    <DocumentContent
      chapter={chapter}
      revision={revision()}
      onObject={() => {}}
      onEvidence={() => {}}
    />
  );
  assert.match(html, /This diagram is unavailable/);
  assert.match(html, /This flow is unavailable/);
});

test('diagram references retain narrative transitions and canonical headings', () => {
  const saved = revision();
  saved.content.model.claims = [
    {
      id: 'claim',
      text: 'A verified boundary',
      basis: 'source_backed',
      evidence_ids: ['evidence'],
    },
  ];
  saved.content.views.diagrams = ['first', 'second'].map((id) => ({
    id,
    title: `Canonical ${id} view`,
    kind: 'data',
    summary: '',
    node_ids: [],
    relation_ids: [],
  }));
  const chapter: Chapter = {
    id: 'data',
    title: 'Data path',
    question: '',
    blocks: [
      { kind: 'diagram_ref', reference_id: 'first' },
      {
        kind: 'markdown',
        markdown: 'The next view explains the output boundary.',
        claim_ids: ['claim'],
      },
      { kind: 'diagram_ref', reference_id: 'second' },
    ],
  };
  const html = renderToStaticMarkup(
    <QueryClientProvider client={new QueryClient()}>
      <DocumentContent
        chapter={chapter}
        revision={saved}
        onObject={() => {}}
        onEvidence={() => {}}
      />
    </QueryClientProvider>
  );
  assert.ok(html.indexOf('Canonical first view') < html.indexOf('The next view'));
  assert.ok(html.indexOf('The next view') < html.indexOf('Canonical second view'));
  assert.match(html, /Supporting evidence \(1\)/);
  assert.match(html, /A verified boundary/);
});

test('saved v1 sections keep their original renderer without a v2 document', () => {
  const saved = revision();
  saved.content.views.sections = [section];
  assert.equal(documentChapters(saved.content.views), undefined);
  assert.deepEqual(legacySections(saved.content.views), [section]);
  const html = renderToStaticMarkup(
    <SectionContent section={section} revision={saved} onObject={() => {}} onEvidence={() => {}} />
  );
  assert.match(html, /Product overview/);
  assert.match(html, /Saved narrative/);
});

test('partial research stays visible with its missing scope and never assumes complete coverage', () => {
  const event = {
    id: 1,
    type: 'research_partial',
    stage: 'researching',
    message: 'Recovery paths still need investigation.',
    created_at: '2026-10-15T00:00:00Z',
  };
  const html = renderToStaticMarkup(
    <PartialResearchNotice
      events={[event, { ...event, id: 2 }]}
      label="This passport has incomplete research"
    />
  );
  assert.match(html, /role="status"/);
  assert.match(html, /This passport has incomplete research/);
  assert.match(html, /Treat this draft as partial/);
  assert.equal(html.match(/Recovery paths still need investigation\./g)?.length, 1);
  assert.equal(
    renderToStaticMarkup(<PartialResearchNotice events={undefined} label="Research warning" />),
    ''
  );
  assert.equal(
    renderToStaticMarkup(
      <PartialResearchNotice
        events={[{ ...event, type: 'stage_completed' }]}
        label="Research warning"
      />
    ),
    ''
  );
});
