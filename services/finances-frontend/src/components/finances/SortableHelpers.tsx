'use client';

import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

export type DndListeners = ReturnType<typeof useSortable>['listeners'];
export type DndAttributes = ReturnType<typeof useSortable>['attributes'];

export function DragHandle({ listeners, attributes }: { listeners?: DndListeners; attributes?: DndAttributes }) {
  return (
    <button
      className="touch-none p-1 cursor-grab active:cursor-grabbing text-white/30 hover:text-white/60 transition-colors"
      {...attributes}
      {...listeners}
    >
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
        <circle cx="9" cy="6" r="1.5" />
        <circle cx="15" cy="6" r="1.5" />
        <circle cx="9" cy="12" r="1.5" />
        <circle cx="15" cy="12" r="1.5" />
        <circle cx="9" cy="18" r="1.5" />
        <circle cx="15" cy="18" r="1.5" />
      </svg>
    </button>
  );
}

export function SortableItem({ id, children }: { id: string; children: (props: { listeners: DndListeners; attributes: DndAttributes }) => React.ReactNode }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    zIndex: isDragging ? 50 : undefined,
    opacity: isDragging ? 0.8 : 1,
  };

  return (
    <div ref={setNodeRef} style={style}>
      {children({ listeners, attributes })}
    </div>
  );
}
