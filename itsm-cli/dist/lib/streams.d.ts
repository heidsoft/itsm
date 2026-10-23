/**
 * A consumer that closes the pipe early (`itsm ... | head`, a vanishing CI log
 * collector) must not abort a run that is already writing business data: the
 * teardown steps still need to execute. Node reports that condition as EPIPE
 * (or a write-after-destroy) on stdout/stderr.
 */
export declare function isClosedStreamError(err: unknown): boolean;
/** Progress output is best-effort; losing the reader is not a business failure. */
export declare function writeStderrLine(message: string): void;
