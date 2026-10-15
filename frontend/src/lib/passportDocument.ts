import { sectionTitles } from './passportSections';
import type { Chapter, Diagram, ProductModel, ProductObject, Revision } from './api';

export function objectTypeLabel(object: ProductObject): string {
  const role = object.architecture_role?.trim();
  return (role === undefined || role === '' ? object.kind : role).replaceAll('_', ' ');
}

export function hasLifecycleStatus(object: Pick<ProductObject, 'kind'>): boolean {
  return object.kind !== 'actor' && object.kind !== 'owner';
}

// A missing document is a saved v1 revision, which keeps its original renderer.
export function documentChapters(views: Revision['content']['views']): Chapter[] | undefined {
  if (views.document?.version !== 2) {
    return undefined;
  }
  return (views.document.chapters ?? []).filter((chapter) =>
    chapter.blocks?.some((block) =>
      block.kind === 'markdown' ? (block.markdown ?? '').trim() !== '' : true
    )
  );
}

export function legacySections(views: Revision['content']['views']) {
  const order = Object.keys(sectionTitles);
  return (views.sections ?? []).toSorted((a, b) => order.indexOf(a.id) - order.indexOf(b.id));
}

const diagramLabels: Record<string, string> = {
  context: 'System context',
  containers: 'Runtime units',
  components: 'Component structure',
  repositories: 'Repository connections',
  domain: 'Domain concepts',
  data: 'Data and artifacts',
  deployment: 'Runtime placement',
  states: 'States and transitions',
};

export function diagramScope(diagram: Diagram, model: ProductModel): string {
  const label = diagramLabels[diagram.kind] ?? diagram.kind.replaceAll('_', ' ');
  const scope = (model.objects ?? []).find((object) => object.id === diagram.scope_object_id);
  return scope === undefined ? label : `${label} · ${scope.name}`;
}
