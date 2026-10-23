/**
 * A consumer that closes the pipe early (`itsm ... | head`, a vanishing CI log
 * collector) must not abort a run that is already writing business data: the
 * teardown steps still need to execute. Node reports that condition as EPIPE
 * (or a write-after-destroy) on stdout/stderr.
 */
export function isClosedStreamError(err) {
    const code = err?.code;
    return code === 'EPIPE' || code === 'ERR_STREAM_DESTROYED' || code === 'ERR_STREAM_WRITE_AFTER_END';
}
/** Progress output is best-effort; losing the reader is not a business failure. */
export function writeStderrLine(message) {
    try {
        process.stderr.write(`${message}\n`);
    }
    catch (err) {
        if (!isClosedStreamError(err))
            throw err;
    }
}
