import { cva, type VariantProps } from 'class-variance-authority';

export const modalOverlayVariants = cva('fixed inset-0', {
	variants: {
		backdrop: {
			dim: 'bg-black/50',
			dark: 'bg-black/80',
			none: ''
		},
		layout: {
			center: 'flex items-center justify-center',
			fullscreen: 'flex flex-col',
			raw: ''
		},
		padding: {
			none: '',
			sm: 'p-2',
			md: 'p-4',
			lg: 'p-6'
		},
		z: {
			40: 'z-40',
			50: 'z-50'
		}
	},
	defaultVariants: {
		backdrop: 'dim',
		layout: 'center',
		padding: 'md',
		z: 50
	}
});

export const modalContentVariants = cva('focus:outline-none', {
	variants: {
		size: {
			sm: 'w-full max-w-sm',
			md: 'w-full max-w-md',
			lg: 'w-full max-w-lg',
			xl: 'w-full max-w-3xl',
			full: 'w-full max-w-full',
			auto: 'w-auto max-w-full'
		}
	},
	defaultVariants: {
		size: 'lg'
	}
});

export type ModalBackdrop = VariantProps<typeof modalOverlayVariants>['backdrop'];
export type ModalLayout = VariantProps<typeof modalOverlayVariants>['layout'];
export type ModalPadding = VariantProps<typeof modalOverlayVariants>['padding'];
export type ModalZ = VariantProps<typeof modalOverlayVariants>['z'];
export type ModalSize = VariantProps<typeof modalContentVariants>['size'];
