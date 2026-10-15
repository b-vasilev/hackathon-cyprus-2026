import { Handle, Position, type Node, type NodeProps } from '@xyflow/react';
import { ArrowUpRight, Boxes, Database, FileStack, Globe2, UserRound } from 'lucide-react';
import { hasLifecycleStatus, objectTypeLabel } from '../lib/passportDocument';
import type { ProductObject } from '../lib/api';
import type { DiagramPort } from '../lib/diagramLayout';

export interface ArchitectureNodeData extends Record<string, unknown> {
  object: ProductObject;
  ports: DiagramPort[];
  onInspect: () => void;
  focused: boolean;
}

export type ArchitectureNode = Node<ArchitectureNodeData, 'architecture'>;

const positions = {
  WEST: Position.Left,
  EAST: Position.Right,
  NORTH: Position.Top,
  SOUTH: Position.Bottom,
};

function KindIcon({ kind }: Readonly<{ kind: string }>) {
  switch (kind) {
    case 'actor':
      return <UserRound size={18} aria-hidden="true" />;
    case 'store':
      return <Database size={18} aria-hidden="true" />;
    case 'entity':
      return <FileStack size={18} aria-hidden="true" />;
    case 'external_system':
      return <Globe2 size={18} aria-hidden="true" />;
    default:
      return <Boxes size={18} aria-hidden="true" />;
  }
}

export function DiagramNode({ data }: NodeProps<ArchitectureNode>) {
  const { object } = data;
  return (
    <div
      className="architecture-node"
      data-kind={object.kind}
      data-status={object.status}
      data-focused={data.focused}
    >
      {data.ports.map((port) => (
        <Handle
          key={port.id}
          id={port.id}
          type={port.target ? 'target' : 'source'}
          position={positions[port.side]}
          style={{ left: port.x ?? 0, top: port.y ?? 0, right: 'auto', bottom: 'auto' }}
          isConnectable={false}
        />
      ))}
      <button
        type="button"
        className="architecture-node-button nodrag nopan"
        onClick={data.onInspect}
        aria-label={`Inspect ${object.name}`}
      >
        <span className="architecture-node-heading">
          <span className="architecture-node-icon">
            <KindIcon kind={object.kind} />
          </span>
          <span className="node-kind">{objectTypeLabel(object)}</span>
          <ArrowUpRight size={15} className="architecture-inspect" aria-hidden="true" />
        </span>
        <strong>{object.name}</strong>
        {object.technology !== '' && (
          <span className="architecture-technology">{object.technology}</span>
        )}
        {object.kind !== 'actor' && (
          <span className="architecture-responsibility">{object.summary}</span>
        )}
        {hasLifecycleStatus(object) && (
          <span className="architecture-status">
            <span aria-hidden="true" />
            {object.status.replaceAll('_', ' ')}
          </span>
        )}
      </button>
    </div>
  );
}
