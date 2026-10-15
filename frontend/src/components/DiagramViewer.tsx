import { useQuery } from '@tanstack/react-query';
import {
  Background,
  Controls,
  Handle,
  MarkerType,
  Position,
  ReactFlow,
  type Node,
  type NodeProps,
} from '@xyflow/react';
import { Maximize2 } from 'lucide-react';
import { useMemo, useState } from 'react';
import { Dialog, ErrorNotice, Loading } from './ui';
import type { Diagram, ProductModel, ProductObject } from '../lib/api';
import '@xyflow/react/dist/style.css';

interface ObjectNodeData extends Record<string, unknown> {
  title: string;
  kind: string;
  technology: string;
}
type ObjectNode = Node<ObjectNodeData, 'productObject'>;

function ProductNode({ data }: NodeProps<ObjectNode>) {
  return (
    <div className="diagram-node">
      <Handle type="target" position={Position.Left} />
      <span className="node-kind">{data.kind.replaceAll('_', ' ')}</span>
      <strong>{data.title}</strong>
      <span className="node-tech">{data.technology}</span>
      <Handle type="source" position={Position.Right} />
    </div>
  );
}

const nodeTypes = { productObject: ProductNode };

export function DiagramViewer({
  diagram,
  model,
  onSelect,
  highlighted = [],
}: Readonly<{
  diagram: Diagram;
  model: ProductModel;
  onSelect: (object: ProductObject) => void;
  highlighted?: string[];
}>) {
  const [expanded, setExpanded] = useState(false);
  const objects = useMemo(
    () => new Map((model.objects ?? []).map((object) => [object.id, object])),
    [model.objects]
  );
  const relations = (model.relations ?? []).filter((relation) =>
    diagram.relation_ids?.includes(relation.id)
  );
  const layout = useQuery({
    queryKey: ['diagram-layout', diagram, model.objects, model.relations],
    queryFn: async () => {
      const { default: Layout } = await import('elkjs/lib/elk.bundled.js');
      const engine = new Layout();
      return engine.layout({
        id: diagram.id,
        layoutOptions: {
          'elk.algorithm': 'layered',
          'elk.direction': 'RIGHT',
          'elk.spacing.nodeNode': '36',
          'elk.layered.spacing.nodeNodeBetweenLayers': '100',
        },
        children: (diagram.node_ids ?? []).map((id) => ({ id, width: 230, height: 104 })),
        edges: relations.map((relation) => ({
          id: relation.id,
          sources: [relation.from],
          targets: [relation.to],
        })),
      });
    },
    staleTime: Number.POSITIVE_INFINITY,
  });
  const nodes: ObjectNode[] = (layout.data?.children ?? []).map((item) => {
    const object = objects.get(item.id);
    return {
      id: item.id,
      type: 'productObject',
      position: { x: item.x ?? 0, y: item.y ?? 0 },
      data: {
        title: object?.name ?? item.id,
        kind: object?.kind ?? '',
        technology: object?.technology ?? '',
      },
      selected: highlighted.includes(item.id),
      ariaLabel: `Inspect ${object?.name ?? item.id}`,
    };
  });
  const edges = relations.map((relation) => ({
    id: relation.id,
    source: relation.from,
    target: relation.to,
    label: relation.label,
    type: 'smoothstep',
    markerEnd: { type: MarkerType.ArrowClosed },
    style: { stroke: 'var(--of-control-border)' },
    labelStyle: { fill: 'var(--of-muted)', fontSize: 12 },
    labelBgStyle: { fill: 'var(--of-surface)' },
  }));
  function graph() {
    return (
      <div className="graph-canvas">
        {layout.isPending ? (
          <Loading label="Arranging diagram…" />
        ) : (
          <ReactFlow
            nodes={nodes}
            edges={edges}
            nodeTypes={nodeTypes}
            fitView
            fitViewOptions={{ padding: 0.2 }}
            minZoom={0.25}
            maxZoom={1.5}
            nodesDraggable={false}
            nodesConnectable={false}
            onNodeClick={(_, node) => {
              const object = objects.get(node.id);
              if (object !== undefined) {
                onSelect(object);
              }
            }}
          >
            <Background color="var(--of-border)" gap={24} size={1} />
            <Controls showInteractive={false} />
          </ReactFlow>
        )}
      </div>
    );
  }
  return (
    <div className="diagram-view">
      <div className="diagram-toolbar">
        <div>
          <span className="eyebrow">{diagram.kind.replaceAll('_', ' ')}</span>
          <h3>{diagram.title}</h3>
        </div>
        <button type="button" className="button quiet" onClick={() => setExpanded(true)}>
          <Maximize2 size={15} />
          Expand
        </button>
      </div>
      {layout.isError && <ErrorNotice error={layout.error} />}
      {graph()}
      <p className="diagram-caption">
        {diagram.summary}
        <span>Select a component to inspect its role and sources.</span>
      </p>
      {expanded && (
        <Dialog title={diagram.title} wide onClose={() => setExpanded(false)}>
          {graph()}
        </Dialog>
      )}
    </div>
  );
}
