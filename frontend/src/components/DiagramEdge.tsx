import { BaseEdge, type Edge, type EdgeProps } from '@xyflow/react';
import { routedPath } from '../lib/diagramLayout';
import type { ElkExtendedEdge } from 'elkjs';

export interface ArchitectureEdgeData extends Record<string, unknown> {
  route: ElkExtendedEdge;
  kind: string;
  focused: boolean;
}

export type ArchitectureEdge = Edge<ArchitectureEdgeData, 'architecture'>;

export function DiagramEdge({ id, data, markerEnd }: EdgeProps<ArchitectureEdge>) {
  if (data === undefined) {
    return null;
  }
  return (
    <g className="architecture-edge" data-kind={data.kind} data-focused={data.focused}>
      {data.route.sections?.map((section) => (
        <BaseEdge
          key={section.id}
          id={`${id}:${section.id}`}
          path={routedPath([section.startPoint, ...(section.bendPoints ?? []), section.endPoint])}
          markerEnd={markerEnd}
        />
      ))}
      {data.route.labels?.map((label) => (
        <foreignObject
          key={label.id}
          x={label.x}
          y={label.y}
          width={label.width}
          height={label.height}
          pointerEvents="none"
        >
          <div className="architecture-edge-label">{label.text}</div>
        </foreignObject>
      ))}
    </g>
  );
}
