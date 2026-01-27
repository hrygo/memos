import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import ForceGraph2D, { ForceGraphMethods, LinkObject, NodeObject } from "react-force-graph-2d";
import { useTranslation } from "react-i18next";
import { Loader2, ZoomIn, ZoomOut, Maximize2, Filter, RefreshCw, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import useNavigateTo from "@/hooks/useNavigateTo";
import { useKnowledgeGraph } from "@/hooks/useAIQueries";
import { CLUSTER_COLORS, EDGE_TYPE_COLORS, convertProtoToGraphData } from "./types";

interface Props {
  className?: string;
}

interface ForceGraphNode extends NodeObject {
  id: string;
  label: string;
  type: string;
  tags: string[];
  importance: number;
  cluster: number;
  createdTs: bigint;
  x?: number;
  y?: number;
}

interface ForceGraphLink extends LinkObject {
  source: string | ForceGraphNode;
  target: string | ForceGraphNode;
  type: string;
  weight: number;
}

const KnowledgeGraphView = ({ className }: Props) => {
  const { t } = useTranslation();
  const navigateTo = useNavigateTo();
  const containerRef = useRef<HTMLDivElement>(null);
  const graphRef = useRef<ForceGraphMethods<ForceGraphNode, ForceGraphLink> | undefined>(undefined);
  const [graphSize, setGraphSize] = useState({ width: 0, height: 0 });

  // Filter state
  const [minImportance, setMinImportance] = useState(0);
  const [selectedClusters, setSelectedClusters] = useState<number[]>([]);
  const [showEdgeTypes, setShowEdgeTypes] = useState<Record<string, boolean>>({
    link: true,
    tag_co: true,
    semantic: true,
  });

  // Fetch graph data
  const {
    data: graphResponse,
    isLoading,
    isError,
    refetch,
  } = useKnowledgeGraph({
    tags: [],
    minImportance,
    clusters: selectedClusters,
  });

  // Convert proto response to graph data
  const graphData = useMemo(() => {
    if (!graphResponse) return null;
    return convertProtoToGraphData(
      graphResponse.nodes,
      graphResponse.edges,
      graphResponse.stats ?? null,
      graphResponse.buildMs
    );
  }, [graphResponse]);

  // Filter edges by type
  const filteredGraphData = useMemo(() => {
    if (!graphData) return { nodes: [], links: [] };

    const filteredEdges = graphData.edges.filter((e) => showEdgeTypes[e.type] !== false);

    // Get node IDs that have at least one edge
    const connectedNodeIds = new Set<string>();
    filteredEdges.forEach((e) => {
      connectedNodeIds.add(e.source);
      connectedNodeIds.add(e.target);
    });

    // Filter nodes to only include connected ones (or all if no edges)
    const filteredNodes =
      connectedNodeIds.size > 0 ? graphData.nodes.filter((n) => connectedNodeIds.has(n.id)) : graphData.nodes;

    return {
      nodes: filteredNodes.map((n) => ({
        id: n.id,
        label: n.label,
        type: n.type,
        tags: n.tags,
        importance: n.importance,
        cluster: n.cluster,
        createdTs: n.createdTs,
      })),
      links: filteredEdges.map((e) => ({
        source: e.source,
        target: e.target,
        type: e.type,
        weight: e.weight,
      })),
    };
  }, [graphData, showEdgeTypes]);

  // Available clusters from data
  const availableClusters = useMemo(() => {
    if (!graphData) return [];
    const clusters = new Set<number>();
    graphData.nodes.forEach((n) => clusters.add(n.cluster));
    return Array.from(clusters).sort((a, b) => a - b);
  }, [graphData]);

  // Update container size
  useEffect(() => {
    if (!containerRef.current) return;

    const updateSize = () => {
      if (containerRef.current) {
        const rect = containerRef.current.getBoundingClientRect();
        setGraphSize({ width: rect.width, height: rect.height });
      }
    };

    updateSize();
    const resizeObserver = new ResizeObserver(updateSize);
    resizeObserver.observe(containerRef.current);

    return () => resizeObserver.disconnect();
  }, []);

  // Node click handler
  const onNodeClick = useCallback(
    (node: ForceGraphNode) => {
      if (node.id.startsWith("memos/")) {
        navigateTo(`/${node.id}`);
      }
    },
    [navigateTo]
  );

  // Node color by cluster
  const getNodeColor = useCallback((node: ForceGraphNode) => {
    return CLUSTER_COLORS[node.cluster % CLUSTER_COLORS.length];
  }, []);

  // Node size by importance
  const getNodeSize = useCallback((node: ForceGraphNode) => {
    return 4 + node.importance * 12;
  }, []);

  // Link color by type
  const getLinkColor = useCallback((link: ForceGraphLink) => {
    return EDGE_TYPE_COLORS[link.type] || "#94a3b8";
  }, []);

  // Link width by weight
  const getLinkWidth = useCallback((link: ForceGraphLink) => {
    return 0.5 + link.weight * 2;
  }, []);

  // Zoom controls
  const handleZoomIn = () => {
    if (graphRef.current) {
      graphRef.current.zoom(graphRef.current.zoom() * 1.5, 300);
    }
  };

  const handleZoomOut = () => {
    if (graphRef.current) {
      graphRef.current.zoom(graphRef.current.zoom() / 1.5, 300);
    }
  };

  const handleFitView = () => {
    if (graphRef.current) {
      graphRef.current.zoomToFit(400, 50);
    }
  };

  // Toggle cluster selection
  const toggleCluster = (cluster: number) => {
    setSelectedClusters((prev) => (prev.includes(cluster) ? prev.filter((c) => c !== cluster) : [...prev, cluster]));
  };

  // Toggle edge type
  const toggleEdgeType = (type: string) => {
    setShowEdgeTypes((prev) => ({ ...prev, [type]: !prev[type] }));
  };

  // Handle importance slider change
  const handleImportanceChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setMinImportance(Number(e.target.value) / 100);
  };

  if (isLoading) {
    return (
      <div className={cn("flex items-center justify-center h-full", className)}>
        <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (isError) {
    return (
      <div className={cn("flex flex-col items-center justify-center h-full gap-3", className)}>
        <AlertCircle className="w-8 h-8 text-destructive" />
        <p className="text-muted-foreground">{t("ai.knowledge-graph.load-error")}</p>
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          <RefreshCw className="w-4 h-4 mr-1" />
          {t("common.retry")}
        </Button>
      </div>
    );
  }

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Toolbar */}
      <div className="flex items-center justify-between p-2 border-b">
        <div className="flex items-center gap-2">
          {/* Filter popover */}
          <Popover>
            <PopoverTrigger asChild>
              <Button variant="outline" size="sm">
                <Filter className="w-4 h-4 mr-1" />
                {t("ai.knowledge-graph.filter")}
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-80">
              <div className="space-y-4">
                {/* Importance slider */}
                <div>
                  <label className="text-sm font-medium">{t("ai.knowledge-graph.min-importance")}</label>
                  <div className="flex items-center gap-2 mt-2">
                    <input
                      type="range"
                      min="0"
                      max="100"
                      step="5"
                      value={minImportance * 100}
                      onChange={handleImportanceChange}
                      className="flex-1 h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700"
                    />
                    <span className="text-sm text-muted-foreground w-10">{Math.round(minImportance * 100)}%</span>
                  </div>
                </div>

                {/* Edge types */}
                <div>
                  <label className="text-sm font-medium">{t("ai.knowledge-graph.edge-types")}</label>
                  <div className="flex flex-col gap-2 mt-2">
                    {Object.entries(EDGE_TYPE_COLORS).map(([type, color]) => (
                      <div key={type} className="flex items-center gap-2">
                        <Checkbox
                          id={`edge-${type}`}
                          checked={showEdgeTypes[type]}
                          onCheckedChange={() => toggleEdgeType(type)}
                        />
                        <div className="w-3 h-3 rounded-full" style={{ backgroundColor: color }} />
                        <label htmlFor={`edge-${type}`} className="text-sm">
                          {t(`ai.knowledge-graph.edge-type-${type}`)}
                        </label>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Clusters */}
                {availableClusters.length > 0 && (
                  <div>
                    <label className="text-sm font-medium">{t("ai.knowledge-graph.clusters")}</label>
                    <div className="flex flex-wrap gap-1 mt-2">
                      {availableClusters.map((cluster) => (
                        <Badge
                          key={cluster}
                          variant={selectedClusters.length === 0 || selectedClusters.includes(cluster) ? "default" : "outline"}
                          className="cursor-pointer"
                          style={{
                            backgroundColor:
                              selectedClusters.length === 0 || selectedClusters.includes(cluster)
                                ? CLUSTER_COLORS[cluster % CLUSTER_COLORS.length]
                                : undefined,
                          }}
                          onClick={() => toggleCluster(cluster)}
                        >
                          {cluster + 1}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </PopoverContent>
          </Popover>

          {/* Refresh button */}
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="w-4 h-4" />
          </Button>
        </div>

        {/* Zoom controls */}
        <div className="flex items-center gap-1">
          <Button variant="ghost" size="icon" onClick={handleZoomOut}>
            <ZoomOut className="w-4 h-4" />
          </Button>
          <Button variant="ghost" size="icon" onClick={handleZoomIn}>
            <ZoomIn className="w-4 h-4" />
          </Button>
          <Button variant="ghost" size="icon" onClick={handleFitView}>
            <Maximize2 className="w-4 h-4" />
          </Button>
        </div>
      </div>

      {/* Graph container */}
      <div ref={containerRef} className="flex-1 relative">
        {filteredGraphData.nodes.length > 0 ? (
          <ForceGraph2D
            ref={graphRef}
            width={graphSize.width}
            height={graphSize.height}
            graphData={filteredGraphData}
            nodeId="id"
            nodeLabel={(node) => `${node.label}\n${node.tags.map((tag: string) => `#${tag}`).join(" ")}`}
            nodeColor={getNodeColor}
            nodeRelSize={1}
            nodeVal={getNodeSize}
            linkColor={getLinkColor}
            linkWidth={getLinkWidth}
            linkDirectionalParticles={1}
            linkDirectionalParticleWidth={(link) => link.weight * 2}
            onNodeClick={onNodeClick}
            cooldownTicks={100}
            enableZoomInteraction
            enablePanInteraction
          />
        ) : (
          <div className="absolute inset-0 flex items-center justify-center">
            <p className="text-muted-foreground">{t("ai.knowledge-graph.no-data")}</p>
          </div>
        )}
      </div>

      {/* Stats footer */}
      {graphData?.stats && (
        <div className="flex items-center justify-between p-2 border-t text-xs text-muted-foreground">
          <div className="flex items-center gap-4">
            <span>
              {t("ai.knowledge-graph.nodes")}: {graphData.stats.nodeCount}
            </span>
            <span>
              {t("ai.knowledge-graph.edges")}: {graphData.stats.edgeCount}
            </span>
            <span>
              {t("ai.knowledge-graph.clusters-count")}: {graphData.stats.clusterCount}
            </span>
          </div>
          <span>
            {t("ai.knowledge-graph.build-time")}: {Number(graphData.buildMs)}ms
          </span>
        </div>
      )}
    </div>
  );
};

export default KnowledgeGraphView;
