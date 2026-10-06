import React, { useState, useEffect, useCallback, useRef } from 'react';
import FullCalendar from '@fullcalendar/react';
import dayGridPlugin from '@fullcalendar/daygrid';
import timeGridPlugin from '@fullcalendar/timegrid';
import interactionPlugin from '@fullcalendar/interaction';
import { DateSelectArg, EventClickArg } from '@fullcalendar/core';
import { Plus, Loader2 } from 'lucide-react';
import { api } from '../api/client';
import { TimeBlock, Task } from '../types';
import { TimeBlockModal } from '../components/TimeBlockModal';

export const CalendarPage: React.FC = () => {
  const [timeBlocks, setTimeBlocks] = useState<TimeBlock[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);

  // Time block modal state
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [selectedBlock, setSelectedBlock] = useState<TimeBlock | null>(null);
  const [slotStart, setSlotStart] = useState<string>('');
  const [slotEnd, setSlotEnd] = useState<string>('');

  const calendarRef = useRef<FullCalendar>(null);

  const fetchCalendarEvents = useCallback(async () => {
    setLoading(true);
    try {
      const now = new Date();
      const fromStr = new Date(now.getFullYear(), now.getMonth() - 1, 1).toISOString();
      const toStr = new Date(now.getFullYear(), now.getMonth() + 2, 0).toISOString();

      const [blocksRes, tasksRes] = await Promise.all([
        api.get<TimeBlock[]>(`/time-blocks?from=${fromStr}&to=${toStr}`),
        api.get<Task[]>('/tasks?limit=100').catch(() => ({ data: [] })),
      ]);

      setTimeBlocks(Array.isArray(blocksRes.data) ? blocksRes.data : []);
      setTasks(Array.isArray(tasksRes.data) ? tasksRes.data : []);
    } catch (err) {
      console.error('Failed to load calendar data:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchCalendarEvents();
  }, [fetchCalendarEvents]);

  const handleDateSelect = (selectInfo: DateSelectArg) => {
    setSelectedBlock(null);
    setSlotStart(selectInfo.startStr);
    setSlotEnd(selectInfo.endStr);
    setIsModalOpen(true);
  };

  const handleEventClick = (clickInfo: EventClickArg) => {
    const blockId = clickInfo.event.extendedProps?.blockId;
    if (blockId) {
      const found = timeBlocks.find((b) => b.id === blockId);
      if (found) {
        setSelectedBlock(found);
        setIsModalOpen(true);
      }
    }
  };

  const calendarEvents = [
    // Focus time blocks (Notion blue badge style)
    ...timeBlocks.map((b) => ({
      id: `block-${b.id}`,
      title: `${b.title}`,
      start: b.starts_at,
      end: b.ends_at,
      backgroundColor: '#1b2b3a',
      borderColor: '#254460',
      textColor: '#70a5e8',
      extendedProps: {
        blockId: b.id,
        type: 'time_block',
      },
    })),
    // Task due dates (Notion orange/yellow badge style)
    ...tasks
      .filter((t) => (t.due_at || t.due_date) && t.status !== 'done')
      .map((t) => ({
        id: `task-${t.id}`,
        title: `Due: ${t.title}`,
        start: (t.due_at || t.due_date)!,
        allDay: true,
        backgroundColor: '#3b2b1d',
        borderColor: '#593f26',
        textColor: '#e59b5f',
        extendedProps: {
          taskId: t.id,
          type: 'task_due',
        },
      })),
  ];

  return (
    <div className="max-w-6xl mx-auto space-y-6 pb-16 font-sans">
      <div className="space-y-1">
        <div className="text-3xl select-none">📅</div>
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold tracking-tight text-[#ebebeb]">
            Calendar
          </h1>
          <button
            onClick={() => {
              setSelectedBlock(null);
              setSlotStart(new Date().toISOString());
              setSlotEnd(new Date(Date.now() + 3600000).toISOString());
              setIsModalOpen(true);
            }}
            className="flex items-center gap-1.5 bg-[#2383e2] hover:bg-[#1d72c5] text-white text-xs font-medium px-3 py-1.5 rounded-md transition-colors cursor-pointer"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>Schedule Block</span>
          </button>
        </div>
      </div>

      <div className="notion-card p-4 relative">
        {loading && (
          <div className="absolute inset-0 bg-[#1e1e1e]/60 z-20 flex items-center justify-center rounded-lg">
            <Loader2 className="w-5 h-5 text-[#888888] animate-spin" />
          </div>
        )}

        <div className="h-[700px]">
          <FullCalendar
            ref={calendarRef}
            plugins={[dayGridPlugin, timeGridPlugin, interactionPlugin]}
            initialView="timeGridWeek"
            headerToolbar={{
              left: 'prev,next today',
              center: 'title',
              right: 'dayGridMonth,timeGridWeek,timeGridDay',
            }}
            selectable={true}
            selectMirror={true}
            dayMaxEvents={true}
            weekends={true}
            nowIndicator={true}
            events={calendarEvents}
            select={handleDateSelect}
            eventClick={handleEventClick}
            slotMinTime="07:00:00"
            slotMaxTime="23:00:00"
            slotDuration="00:30:00"
            allDaySlot={true}
            height="100%"
          />
        </div>
      </div>

      <TimeBlockModal
        isOpen={isModalOpen}
        timeBlock={selectedBlock}
        initialStart={slotStart}
        initialEnd={slotEnd}
        onClose={() => {
          setIsModalOpen(false);
          setSelectedBlock(null);
        }}
        onSuccess={() => {
          setIsModalOpen(false);
          setSelectedBlock(null);
          fetchCalendarEvents();
        }}
      />
    </div>
  );
};
