import { Check, CircleAlert, Clock3 } from 'lucide-react';
import { duration } from '../lib/generation';
import type { Run } from '../lib/api';
import './CompositionProgress.css';

type Composition = NonNullable<Run['composition']>;
type Task = NonNullable<Composition['tasks']>[number];

function TaskIcon({ status }: Readonly<{ status: string }>) {
  let Icon = Clock3;
  if (status === 'completed') {
    Icon = Check;
  } else if (status === 'failed') {
    Icon = CircleAlert;
  }
  return <Icon size={16} aria-hidden="true" />;
}

function taskStatus(task: Task, running: boolean): string {
  if (task.status === 'completed') {
    return 'Completed';
  }
  if (task.status === 'failed') {
    return 'Failed';
  }
  if (task.status === 'stopped') {
    return 'Stopped';
  }
  if (!running) {
    return task.status === 'queued' ? 'Not started' : 'Stopped';
  }
  if (task.status === 'queued') {
    return 'Queued';
  }
  return task.attempt > 1 ? 'Retrying' : 'In progress';
}

export function CompositionProgress({
  composition,
  now,
  running,
}: Readonly<{ composition: Composition; now: number; running: boolean }>) {
  const tasks = composition.tasks ?? [];
  const completed = tasks.filter((task) => task.status === 'completed').length;
  const active = tasks.filter((task) => task.status === 'running').length;
  const queued = tasks.filter((task) => task.status === 'queued').length;
  const failed = tasks.filter((task) => task.status === 'failed').length;
  const chapters = [...new Set(tasks.map((task) => task.chapter_id))];
  let summary = 'Writing stopped';
  if (running) {
    summary = `${active} in progress · ${queued} queued · up to ${composition.parallelism} requests at once`;
  } else if (tasks.length > 0 && completed === tasks.length) {
    summary = 'Writing complete';
  }
  if (failed > 0) {
    summary += ` · ${failed} failed`;
  }
  return (
    <section className="composition-progress" aria-label="Passport writing tasks">
      <div className="composition-heading">
        <h3>Writing the passport in small parts</h3>
        <span>
          {completed} of {tasks.length} current tasks completed
        </span>
      </div>
      <p className="composition-summary" role="status">
        {summary}
      </p>
      {composition.action !== '' && <p className="composition-action">{composition.action}</p>}
      <p className="meta">
        Each chapter gets a short plan, then its points, tables and diagrams are written separately.
        Overview follows the detailed chapters.
      </p>
      {chapters.map((chapter) => {
        const chapterTasks = tasks.filter((task) => task.chapter_id === chapter);
        const done = chapterTasks.filter((task) => task.status === 'completed').length;
        const needsAttention = chapterTasks.some(
          (task) => task.status === 'running' || task.status === 'failed'
        );
        return (
          <details className="composition-chapter" key={chapter} open={needsAttention}>
            <summary>
              {chapter === '' ? 'Product card' : chapter.replaceAll('_', ' ')} · {done}/
              {chapterTasks.length} completed
            </summary>
            <ul>
              {chapterTasks.map((task) => (
                <li key={task.id} data-status={task.status}>
                  <TaskIcon status={task.status} />
                  <div>
                    <strong>{task.title}</strong>
                    <span>
                      {taskStatus(task, running)}
                      {task.attempt > 0 && ` · attempt ${task.attempt}/2`}
                      {task.started_at !== '' &&
                        ` · ${duration(task.started_at, task.finished_at === '' ? now : Date.parse(task.finished_at))}`}
                    </span>
                  </div>
                </li>
              ))}
            </ul>
          </details>
        );
      })}
    </section>
  );
}
