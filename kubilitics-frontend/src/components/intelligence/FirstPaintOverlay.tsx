// src/components/intelligence/FirstPaintOverlay.tsx

import { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';

interface FirstPaintOverlayProps {
  resourceName: string;
  kind: string;
}

export function FirstPaintOverlay({ resourceName, kind }: FirstPaintOverlayProps) {
  const [visible, setVisible] = useState(true);

  useEffect(() => {
    const timer = setTimeout(() => setVisible(false), 1500);
    return () => clearTimeout(timer);
  }, []);

  return (
    <AnimatePresence>
      {visible && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.4 }}
          className="absolute inset-0 z-10 flex items-center justify-center pointer-events-none"
        >
          <div className="bg-black/50 backdrop-blur-sm text-white px-6 py-3 rounded-xl text-sm font-medium">
            Showing impact of <span className="font-bold">{kind}/{resourceName}</span>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
