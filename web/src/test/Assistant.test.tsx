import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ProposalCard } from '../components/ProposalCard';
import { SuggestedPrompts } from '../components/SuggestedPrompts';
import { Proposal } from '../types';

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('Assistant UI Components', () => {
  it('renders ProposalCard with pending status and action buttons', () => {
    const mockProposal: Proposal = {
      id: 'prop-123',
      user_id: 'user-456',
      summary: 'Staged 1 task for your approval',
      status: 'pending',
      created_at: new Date().toISOString(),
      actions: [
        {
          id: 'act-1',
          tool: 'create_task',
          input: {
            title: 'Complete Distributed Systems Lab',
            priority: 1,
            estimate_minutes: 60,
          },
          status: 'pending',
        },
      ],
    };

    render(<ProposalCard proposal={mockProposal} />, { wrapper: createWrapper() });

    expect(screen.getByText('Plan Proposal')).toBeInTheDocument();
    expect(screen.getByText('Awaiting Approval')).toBeInTheDocument();
    expect(screen.getByText('Staged 1 task for your approval')).toBeInTheDocument();
    expect(screen.getByText(/Complete Distributed Systems Lab/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Approve & Apply/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Reject/i })).toBeInTheDocument();
  });

  it('renders persona-tailored SuggestedPrompts and triggers onSelectPrompt', () => {
    const onSelect = vi.fn();
    render(<SuggestedPrompts persona="student" onSelectPrompt={onSelect} />);

    expect(screen.getByText(/suggested for student space/i)).toBeInTheDocument();
    expect(screen.getByText('Plan Exam Revision')).toBeInTheDocument();

    const promptBtn = screen.getByText('Plan Exam Revision');
    fireEvent.click(promptBtn);

    expect(onSelect).toHaveBeenCalledWith(
      expect.stringContaining('revision schedule')
    );
  });
});
