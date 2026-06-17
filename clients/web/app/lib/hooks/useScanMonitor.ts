import { useEffect, useState, useRef } from 'react';
import { createScanJobMonitor } from '../stores/scan-monitor';
import type { StatusMonitorSnapshot, ReportMonitorSnapshot } from '../stores/scan-monitor/types';

export function useScanStatus(jobId: string) {
	const [snapshot, setSnapshot] = useState<StatusMonitorSnapshot | null>(null);

	useEffect(() => {
		if (!jobId) return;

		const monitor = createScanJobMonitor({
			kind: 'status',
			jobId
		}) as any;

		const unsubscribe = monitor.subscribe((newSnapshot: any) => {
			setSnapshot({ ...newSnapshot });
		});

		monitor.start();

		return () => {
			monitor.stop();
			unsubscribe();
		};
	}, [jobId]);

	return {
		status: snapshot?.status ?? 'loading',
		result: snapshot?.job ?? null,
		elapsed: snapshot?.elapsedSeconds ?? 0,
		logs: snapshot?.logs ?? [],
		transport: snapshot?.transport ?? 'idle',
		error: snapshot?.error ?? null
	};
}

export function useScanReport(jobId: string) {
	const [snapshot, setSnapshot] = useState<ReportMonitorSnapshot | null>(null);
	const monitorRef = useRef<any>(null);

	useEffect(() => {
		if (!jobId) return;

		const monitor = createScanJobMonitor({
			kind: 'report',
			jobId
		}) as any;
		monitorRef.current = monitor;

		const unsubscribe = monitor.subscribe((newSnapshot: any) => {
			setSnapshot({ ...newSnapshot });
		});

		monitor.start();

		return () => {
			monitor.stop();
			unsubscribe();
			monitorRef.current = null;
		};
	}, [jobId]);

	const refreshArtifacts = () => {
		if (monitorRef.current) {
			void monitorRef.current.refreshArtifacts();
		}
	};

	return {
		status: snapshot?.status ?? 'loading',
		job: snapshot?.job ?? null,
		report: snapshot?.report ?? null,
		screenshots: snapshot?.screenshots ?? [],
		logs: snapshot?.logs ?? [],
		transport: snapshot?.transport ?? 'idle',
		error: snapshot?.error ?? null,
		refreshArtifacts
	};
}
