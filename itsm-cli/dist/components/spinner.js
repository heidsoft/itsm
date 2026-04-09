import React from 'react';
import { Text } from 'ink';
export const Spinner = ({ text = 'Loading...' }) => {
    const [frame, setFrame] = React.useState(0);
    const frames = ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'];
    React.useEffect(() => {
        const timer = setInterval(() => {
            setFrame(f => (f + 1) % frames.length);
        }, 80);
        return () => clearInterval(timer);
    }, []);
    return (React.createElement(Text, { color: "cyan" },
        frames[frame],
        " ",
        text));
};
