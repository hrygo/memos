import { GraphNode, GraphEdge, GraphStats } from "@/types/proto/api/v1/ai_service_pb";

export interface GraphNodeData {
  id: string;
  label: string;
  type: string;
  tags: string[];
  importance: number;
  cluster: number;
  createdTs: bigint;
}

export interface GraphEdgeData {
  source: string;
  target: string;
  type: string;
  weight: number;
}

export interface GraphData {
  nodes: GraphNodeData[];
  edges: GraphEdgeData[];
  stats: GraphStats | null;
  buildMs: bigint;
}

export interface GraphFilter {
  tags: string[];
  minImportance: number;
  clusters: number[];
}

// Color palettes for clusters
export const CLUSTER_COLORS = [
  "#3b82f6", // blue
  "#10b981", // emerald
  "#f59e0b", // amber
  "#ef4444", // red
  "#8b5cf6", // violet
  "#ec4899", // pink
  "#06b6d4", // cyan
  "#84cc16", // lime
  "#f97316", // orange
  "#6366f1", // indigo
];

// Edge type colors
export const EDGE_TYPE_COLORS: Record<string, string> = {
  link: "#3b82f6", // blue - explicit links
  tag_co: "#10b981", // emerald - tag co-occurrence
  semantic: "#8b5cf6", // violet - semantic similarity
};

export function convertProtoToGraphData(
  nodes: GraphNode[],
  edges: GraphEdge[],
  stats: GraphStats | null,
  buildMs: bigint
): GraphData {
  return {
    nodes: nodes.map((n) => ({
      id: n.id,
      label: n.label,
      type: n.type,
      tags: n.tags,
      importance: n.importance,
      cluster: n.cluster,
      createdTs: n.createdTs,
    })),
    edges: edges.map((e) => ({
      source: e.source,
      target: e.target,
      type: e.type,
      weight: e.weight,
    })),
    stats,
    buildMs,
  };
}
