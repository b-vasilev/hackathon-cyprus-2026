import { useQuery } from '@tanstack/react-query';
import { Controls, MarkerType, ReactFlow, useReactFlow, useStore } from '@xyflow/react';
import { Maximize2, MousePointer2 } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { layoutArchitecture } from '../lib/diagramLayout';
import { diagramScope } from '../lib/passportDocument';
import { DiagramEdge, type ArchitectureEdge } from './DiagramEdge';
import { DiagramNode, type ArchitectureNode } from './DiagramNode';
import { Dialog, DisclosureHint, ErrorNotice, Loading } from './ui';
import type { Diagram, ProductModel, ProductObject } from '../lib/api';

const nodeTypes = { architecture: DiagramNode };
const edgeTypes = { architecture: DiagramEdge };

function FitArchitecture() {
  const { fitView, viewportInitialized } = useReactFlow();
  const width = useStore((state) => state.width);
  const height = useStore((state) => state.height);
  useEffect(() => {
    if (viewportInitialized && width > 0 && height > 0) {
      const frame = requestAnimationFrame(() => {
        void fitView({ padding: 0.06, maxZoom: 1 });
      });
      return () => cancelAnimationFrame(frame);
    }
    return undefined;
  }, [fitView, viewportInitialized, width, height]);
  return null;
}

export function ArchitectureDiagram({
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
  const [hovered, setHovered] = useState('');
  const figure = useRef<HTMLElement>(null);
  const expandedFigure = useRef<HTMLDivElement>(null);
  const [vertical, setVertical] = useState(false);
  useEffect(() => {
    const container = expanded ? expandedFigure.current : figure.current;
    if (container === null) {
      return;
    }
    const observer = new ResizeObserver(([entry]) => {
      if (entry !== undefined) {
        setVertical(entry.contentRect.width <= 700);
      }
    });
    observer.observe(container);
    return () => observer.disconnect();
  }, [expanded]);
  const objects = useMemo(
    () => new Map((model.objects ?? []).map((object) => [object.id, object])),
    [model.objects]
  );
  const layout = useQuery({
    queryKey: ['architecture-layout-v2', diagram, model, vertical],
    queryFn: () => layoutArchitecture(diagram, model, vertical),
    staleTime: Number.POSITIVE_INFINITY,
  });
  const relations = new Map((model.relations ?? []).map((relation) => [relation.id, relation]));
  const focused = new Set(highlighted);
  if (hovered !== '') {
    focused.add(hovered);
  }
  const adjacent = new Set(focused);
  const edges: ArchitectureEdge[] = (layout.data?.edges ?? []).map((route) => {
    const relation = relations.get(route.id);
    const active =
      relation !== undefined && (focused.has(relation.from) || focused.has(relation.to));
    if (active) {
      adjacent.add(relation.from);
      adjacent.add(relation.to);
    }
    return {
      id: route.id,
      type: 'architecture',
      source: relation?.from ?? '',
      target: relation?.to ?? '',
      sourceHandle: route.sources.at(0),
      targetHandle: route.targets.at(0),
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: 16,
        height: 16,
        color: active ? 'var(--product-content-accent, var(--of-brand))' : 'var(--of-muted)',
      },
      data: { route, kind: relation?.kind ?? '', focused: active },
    };
  });
  const nodes: ArchitectureNode[] = (layout.data?.nodes ?? []).flatMap((node) => {
    const object = objects.get(node.id);
    return object === undefined
      ? []
      : [
          {
            id: node.id,
            type: 'architecture',
            position: { x: node.x ?? 0, y: node.y ?? 0 },
            width: node.width,
            height: node.height,
            data: {
              object,
              ports: node.ports,
              onInspect: () => onSelect(object),
              focused: adjacent.has(node.id),
            },
          },
        ];
  });
  function graph() {
    if (layout.isPending) {
      return <Loading label="Arranging architecture…" />;
    }
    if (layout.isError) {
      return <ErrorNotice error={layout.error} />;
    }
    return (
      <div className="architecture-canvas">
        <ReactFlow<ArchitectureNode, ArchitectureEdge>
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          nodesDraggable={false}
          nodesConnectable={false}
          nodesFocusable={false}
          edgesFocusable={false}
          elementsSelectable={false}
          minZoom={0.3}
          maxZoom={1.6}
          zoomOnScroll={false}
          preventScrolling={false}
          onNodeMouseEnter={(_, node) => setHovered(node.id)}
          onNodeMouseLeave={() => setHovered('')}
          aria-label={diagram.title}
        >
          <Controls showInteractive={false} position="bottom-right" />
          <FitArchitecture />
        </ReactFlow>
      </div>
    );
  }
  return (
    <figure className="architecture-diagram" id={`diagram-${diagram.id}`} ref={figure}>
      <div className="architecture-toolbar">
        <div>
          <span className="eyebrow">{diagramScope(diagram, model)}</span>
          <h3>{diagram.title}</h3>
          {diagram.question !== undefined && diagram.question !== '' && (
            <p className="diagram-question">{diagram.question}</p>
          )}
        </div>
        <button type="button" className="button quiet" onClick={() => setExpanded(true)}>
          <Maximize2 size={15} />
          Expand diagram
        </button>
      </div>
      <div className="architecture-frame">
        <div className="architecture-frame-label">{diagramScope(diagram, model)}</div>
        {graph()}
      </div>
      <figcaption className="architecture-caption">
        <div className="architecture-legend">
          <span>
            <i />
            Call / interaction
          </span>
          <span>
            <i className="data" />
            Data / artifact transfer
          </span>
          <span>
            <MousePointer2 size={13} />
            Select a block to inspect
          </span>
        </div>
        {diagram.summary.trim() !== '' && (
          <details className="diagram-explanation">
            <summary>
              About this diagram
              <DisclosureHint show="Show explanation" hide="Hide explanation" />
            </summary>
            <p>{diagram.summary}</p>
          </details>
        )}
      </figcaption>
      {expanded && (
        <Dialog title={diagram.title} wide onClose={() => setExpanded(false)}>
          <div className="architecture-expanded" ref={expandedFigure}>
            {graph()}
          </div>
          <p className="field-help">
            Drag to explore. Select a block for its responsibility, contracts and evidence.
          </p>
        </Dialog>
      )}
    </figure>
  );
}
