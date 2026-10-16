import { useQuery } from '@tanstack/react-query';
import {
  Background,
  Controls,
  Handle,
  MarkerType,
  Position,
  ReactFlow,
  useReactFlow,
  useStore,
  type Node,
  type NodeProps,
} from '@xyflow/react';
import { Maximize2 } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { diagramScope, hasLifecycleStatus, objectTypeLabel } from '../lib/passportDocument';
import { ArchitectureDiagram } from './ArchitectureDiagram';
import { Dialog, ErrorNotice, Loading } from './ui';
import type { Diagram, ProductModel, ProductObject } from '../lib/api';
import '@xyflow/react/dist/style.css';

interface ObjectNodeData extends Record<string, unknown> {
  title: string;
  kind: string;
  technology: string;
  status: string;
}
type ObjectNode = Node<ObjectNodeData, 'productObject'>;

function ProductNode({ data }: NodeProps<ObjectNode>) {
  return (
    <div className="diagram-node" data-status={data.status}>
      <Handle type="target" position={Position.Left} />
      <span className="node-kind">{data.kind}</span>
      <strong>{data.title}</strong>
      <span className="node-tech">{data.technology}</span>
      {data.status !== '' && (
        <span className="node-status">{data.status.replaceAll('_', ' ')}</span>
      )}
      <Handle type="source" position={Position.Right} />
    </div>
  );
}

const nodeTypes = { productObject: ProductNode };

function FitDiagram() {
  const { fitView, viewportInitialized } = useReactFlow();
  const width = useStore((state) => state.width);
  const height = useStore((state) => state.height);
  useEffect(() => {
    if (viewportInitialized && width > 0 && height > 0) {
      const frame = requestAnimationFrame(() => {
        void fitView({ padding: 0.2 });
      });
      return () => cancelAnimationFrame(frame);
    }
    return undefined;
  }, [viewportInitialized, width, height, fitView]);
  return null;
}

export function DiagramViewer(props: Parameters<typeof StandardDiagramViewer>[0]) {
  return ['context', 'containers', 'components', 'repositories'].includes(props.diagram.kind) ? (
    <ArchitectureDiagram {...props} />
  ) : (
    <StandardDiagramViewer {...props} />
  );
}

function StandardDiagramViewer({
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
      const { layoutGraph } = await import('../lib/elkLayout');
      return layoutGraph({
        id: diagram.id,
        layoutOptions: {
          'elk.algorithm': 'layered',
          'elk.direction': 'RIGHT',
          'elk.spacing.nodeNode': '36',
          'elk.layered.spacing.nodeNodeBetweenLayers': '100',
        },
        children: (diagram.node_ids ?? []).map((id) => ({ id, width: 230, height: 128 })),
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
      width: 230,
      height: 128,
      position: { x: item.x ?? 0, y: item.y ?? 0 },
      data: {
        title: object?.name ?? item.id,
        kind: object === undefined ? '' : objectTypeLabel(object),
        technology: object?.technology ?? '',
        status: object !== undefined && hasLifecycleStatus(object) ? object.status : '',
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
    markerEnd: {
      type: MarkerType.ArrowClosed,
      color: 'var(--product-content-accent, var(--of-control-border))',
    },
    style: {
      stroke: 'var(--product-content-accent, var(--of-control-border))',
      strokeDasharray: relation.kind === 'data' ? '8 4' : undefined,
    },
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
            minZoom={0.25}
            maxZoom={1.5}
            nodesDraggable={false}
            nodesConnectable={false}
            zoomOnScroll={false}
            preventScrolling={false}
            onNodeClick={(_, node) => {
              const object = objects.get(node.id);
              if (object !== undefined) {
                onSelect(object);
              }
            }}
          >
            <Background color="var(--of-border)" gap={24} size={1} />
            <Controls showInteractive={false} />
            <FitDiagram />
          </ReactFlow>
        )}
      </div>
    );
  }
  return (
    <div className="diagram-view">
      <div className="diagram-toolbar">
        <div>
          <span className="eyebrow">{diagramScope(diagram, model)}</span>
          <h3>{diagram.title}</h3>
          {diagram.question !== undefined && diagram.question !== '' && (
            <p className="diagram-question">{diagram.question}</p>
          )}
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
      <p className="diagram-legend">
        Solid arrow: call or relation · Dashed arrow: data transfer · Dashed border: planned
      </p>
      {expanded && (
        <Dialog title={diagram.title} wide onClose={() => setExpanded(false)}>
          {graph()}
        </Dialog>
      )}
    </div>
  );
}
