import {
  isValidTransition,
  getTransitionAction,
  getAllowedTransitions,
  validateTransitionOrThrow,
  isFinalStatus,
  isActiveStatus,
  isValidIncidentTransition,
  isValidChangeTransition,
  InvalidStateTransitionError,
} from '../workflow-state-machine';
import { TicketStatus } from '@/constants/taxonomy';

describe('isValidTransition', () => {
  it('allows NEW -> OPEN', () => {
    expect(isValidTransition(TicketStatus.NEW, TicketStatus.OPEN)).toBe(true);
  });

  it('allows NEW -> CANCELLED', () => {
    expect(isValidTransition(TicketStatus.NEW, TicketStatus.CANCELLED)).toBe(true);
  });

  it('disallows NEW -> CLOSED', () => {
    expect(isValidTransition(TicketStatus.NEW, TicketStatus.CLOSED)).toBe(false);
  });

  it('disallows CLOSED -> anything', () => {
    expect(isValidTransition(TicketStatus.CLOSED, TicketStatus.OPEN)).toBe(false);
  });

  it('allows RESOLVED -> CLOSED', () => {
    expect(isValidTransition(TicketStatus.RESOLVED, TicketStatus.CLOSED)).toBe(true);
  });

  it('allows RESOLVED -> OPEN (reopen)', () => {
    expect(isValidTransition(TicketStatus.RESOLVED, TicketStatus.OPEN)).toBe(true);
  });
});

describe('getTransitionAction', () => {
  it('returns resolveTicket for open -> resolved', () => {
    expect(getTransitionAction(TicketStatus.OPEN, TicketStatus.RESOLVED)).toBe('resolveTicket');
  });

  it('returns closeTicket for resolved -> closed', () => {
    expect(getTransitionAction(TicketStatus.RESOLVED, TicketStatus.CLOSED)).toBe('closeTicket');
  });

  it('returns null for unmapped transitions', () => {
    expect(getTransitionAction(TicketStatus.NEW, TicketStatus.OPEN)).toBeNull();
  });
});

describe('getAllowedTransitions', () => {
  it('returns array of allowed statuses from OPEN', () => {
    const allowed = getAllowedTransitions(TicketStatus.OPEN);
    expect(allowed).toContain(TicketStatus.IN_PROGRESS);
    expect(allowed).toContain(TicketStatus.RESOLVED);
  });

  it('returns empty array for CLOSED', () => {
    expect(getAllowedTransitions(TicketStatus.CLOSED)).toEqual([]);
  });
});

describe('validateTransitionOrThrow', () => {
  it('does not throw for valid transition', () => {
    expect(() => validateTransitionOrThrow(TicketStatus.NEW, TicketStatus.OPEN)).not.toThrow();
  });

  it('throws InvalidStateTransitionError for invalid', () => {
    expect(() => validateTransitionOrThrow(TicketStatus.CLOSED, TicketStatus.OPEN)).toThrow(
      InvalidStateTransitionError
    );
  });
});

describe('isFinalStatus / isActiveStatus', () => {
  it('CLOSED is final', () => {
    expect(isFinalStatus(TicketStatus.CLOSED)).toBe(true);
    expect(isActiveStatus(TicketStatus.CLOSED)).toBe(false);
  });

  it('CANCELLED is final', () => {
    expect(isFinalStatus(TicketStatus.CANCELLED)).toBe(true);
  });

  it('OPEN is active', () => {
    expect(isActiveStatus(TicketStatus.OPEN)).toBe(true);
    expect(isFinalStatus(TicketStatus.OPEN)).toBe(false);
  });
});

describe('isValidIncidentTransition', () => {
  it('allows new -> investigating', () => {
    expect(isValidIncidentTransition('new', 'investigating')).toBe(true);
  });
  it('disallows closed -> new', () => {
    expect(isValidIncidentTransition('closed', 'new')).toBe(false);
  });
});

describe('isValidChangeTransition', () => {
  it('allows draft -> pending', () => {
    expect(isValidChangeTransition('draft', 'pending')).toBe(true);
  });
  it('disallows completed -> draft', () => {
    expect(isValidChangeTransition('completed', 'draft')).toBe(false);
  });
});
