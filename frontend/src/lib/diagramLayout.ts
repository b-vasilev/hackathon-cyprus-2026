import type { Diagram, ProductModel } from './api';
import type { ElkExtendedEdge, ElkNode, ElkPoint, ElkPort } from 'elkjs';

export interface DiagramPort extends ElkPort {
  side: 'WEST' | 'EAST' | 'NORTH' | 'SOUTH';
  target: boolean;
}

export interface PlacedNode extends ElkNode {
  ports: DiagramPort[];
}

export interface ArchitectureLayout {
  nodes: PlacedNode[];
  edges: ElkExtendedEdge[];
  width: number;
  height: number;
}

function partitionFor(kind: string): number {
  if (kind === 'actor') {
    return 0;
  }
  return kind === 'external_system' ? 2 : 1;
}

function portSide(incoming: boolean, vertical: boolean): DiagramPort['side'] {
  if (vertical) {
    return incoming ? 'NORTH' : 'SOUTH';
  }
  return incoming ? 'WEST' : 'EAST';
}

function labelLines(text: string): string[] {
  const lines: string[] = [];
  for (const word of text.split(/\s+/)) {
    const previous = lines.at(-1);
    if (previous !== undefined && `${previous} ${word}`.length <= 27) {
      lines[lines.length - 1] = `${previous} ${word}`;
    } else {
      lines.push(word);
    }
  }
  return lines;
}

// ELK owns both node placement and edge routing; the renderer uses its geometry.
export async function layoutArchitecture(
  diagram: Diagram,
  model: ProductModel,
  vertical = false
): Promise<ArchitectureLayout> {
  await document.fonts.ready;
  const { default: ELK } = await import('elkjs/lib/elk.bundled.js');
  const objects = (model.objects ?? []).filter((object) => diagram.node_ids?.includes(object.id));
  const relations = (model.relations ?? []).filter((relation) =>
    diagram.relation_ids?.includes(relation.id)
  );
  const partition = new Map(objects.map((object) => [object.id, partitionFor(object.kind)]));
  const children = objects.map((object) => {
    const ports: DiagramPort[] = relations.flatMap((relation) => {
      if (relation.from !== object.id && relation.to !== object.id) {
        return [];
      }
      const target = relation.to === object.id;
      const forward = (partition.get(relation.from) ?? 1) <= (partition.get(relation.to) ?? 1);
      const incoming = target === forward;
      const side = portSide(incoming, vertical);
      return [
        {
          id: `${object.id}:${relation.id}:${target ? 'in' : 'out'}`,
          width: 0,
          height: 0,
          side,
          target,
          layoutOptions: { 'elk.port.side': side },
        },
      ];
    });
    return {
      id: object.id,
      width: object.kind === 'actor' ? 200 : 280,
      height: object.kind === 'actor' ? 150 : 206,
      ports,
      layoutOptions: {
        'elk.portConstraints': 'FIXED_SIDE',
        'elk.partitioning.partition': String(partition.get(object.id) ?? 1),
      },
    };
  });
  const edges = relations.map((relation) => {
    const lines = labelLines(relation.label);
    return {
      id: relation.id,
      sources: [`${relation.from}:${relation.id}:out`],
      targets: [`${relation.to}:${relation.id}:in`],
      labels: [
        {
          id: `${relation.id}:label`,
          text: lines.join('\n'),
          width: Math.min(204, Math.max(...lines.map((line) => line.length), 1) * 7 + 20),
          height: lines.length * 18 + 12,
        },
      ],
    };
  });
  const graph: ElkNode = {
    id: diagram.id,
    layoutOptions: {
      'elk.algorithm': 'layered',
      'elk.direction': vertical ? 'DOWN' : 'RIGHT',
      'elk.edgeRouting': 'ORTHOGONAL',
      'elk.partitioning.activate': 'true',
      'elk.padding': '[top=32,left=24,bottom=32,right=24]',
      'elk.spacing.nodeNode': '32',
      'elk.spacing.edgeNode': '24',
      'elk.layered.spacing.nodeNodeBetweenLayers': '68',
      'elk.layered.spacing.edgeNodeBetweenLayers': '28',
      'elk.layered.spacing.edgeEdgeBetweenLayers': '24',
      'elk.edgeLabels.inline': 'true',
      'elk.layered.edgeLabels.placement': 'CENTER',
      'elk.layered.nodePlacement.strategy': 'NETWORK_SIMPLEX',
    },
    children,
    edges,
  };
  const result = await new ELK().layout(graph);
  return {
    nodes: (result.children ?? []) as PlacedNode[],
    edges: result.edges ?? [],
    width: result.width ?? 0,
    height: result.height ?? 0,
  };
}

// Round routed bends without replacing ELK's collision-aware route.
export function routedPath(points: ElkPoint[], radius = 8): string {
  const first = points.at(0);
  if (first === undefined) {
    return '';
  }
  let path = `M ${first.x} ${first.y}`;
  for (let index = 1; index < points.length - 1; index++) {
    const previous = points[index - 1];
    const current = points[index];
    const next = points[index + 1];
    const before = Math.hypot(current.x - previous.x, current.y - previous.y);
    const after = Math.hypot(next.x - current.x, next.y - current.y);
    if (before === 0 || after === 0) {
      continue;
    }
    const round = Math.min(radius, before / 2, after / 2);
    const x1 = current.x + ((previous.x - current.x) * round) / before;
    const y1 = current.y + ((previous.y - current.y) * round) / before;
    const x2 = current.x + ((next.x - current.x) * round) / after;
    const y2 = current.y + ((next.y - current.y) * round) / after;
    path += ` L ${x1} ${y1} Q ${current.x} ${current.y} ${x2} ${y2}`;
  }
  const last = points.at(-1);
  if (last !== undefined && points.length > 1) {
    path += ` L ${last.x} ${last.y}`;
  }
  return path;
}
