import { expect, it, vi } from 'vitest';
import { act, fireEvent, render, renderHook, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  Button,
  Input,
  Separator,
  Skeleton,
  Picture,
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
  DialogClose,
  Sheet,
  SheetTrigger,
  SheetContent,
  SheetHeader,
  SheetFooter,
  SheetTitle,
  SheetDescription,
  SheetClose,
  Tooltip,
  TooltipProvider,
  TooltipTrigger,
  TooltipContent,
} from '@/shared/ui';
import { UserCard } from '@/entities/user';
import { getUserInitials, getUserRoleLabel } from '@/entities/user/model/helpers';
import { isMentor, isTrainee } from '@/entities/user/model/types';
import { useIsMobile } from '@/shared/hooks';
import { RequestState } from '@/shared/ui/RequestState';
import { intern, json, mockApi, mount } from './support';

it('forwards native input/button behavior including disabled and readonly controls', () => {
  const click = vi.fn(),
    change = vi.fn();
  const view = render(
    <>
      <Input aria-label="Имя" onChange={change} />
      <Button onClick={click}>Сохранить</Button>
      <Separator />
      <Skeleton aria-label="Загрузка" aria-busy="true" />
    </>,
  );
  fireEvent.change(screen.getByRole('textbox'), { target: { value: 'Анна' } });
  expect(change).toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button'));
  expect(click).toHaveBeenCalledTimes(1);
  expect(screen.getByRole('separator')).toHaveAttribute('aria-orientation', 'horizontal');
  expect(screen.getByLabelText('Загрузка')).toHaveAttribute('aria-busy', 'true');
  view.rerender(
    <>
      <Input aria-label="Имя" readOnly value="Анна" />
      <Button variant="outline" size="sm" disabled onClick={click}>
        Сохранить
      </Button>
      <Separator orientation="vertical" />
    </>,
  );
  expect(screen.getByRole('textbox')).toHaveAttribute('readonly');
  fireEvent.click(screen.getByRole('button'));
  expect(click).toHaveBeenCalledTimes(1);
  expect(screen.getByRole('separator')).toHaveAttribute('aria-orientation', 'vertical');
});
it('sets image loading priority for prominent and deferred images', () => {
  render(
    <>
      <Picture src="/hero.svg" alt="Основное изображение" priority />
      <Picture src="/detail.svg" alt="Детали" width={100} />
    </>,
  );
  expect(screen.getByAltText('Основное изображение')).toHaveAttribute('loading', 'eager');
  expect(screen.getByAltText('Основное изображение')).toHaveAttribute('decoding', 'sync');
  expect(screen.getByAltText('Детали')).toHaveAttribute('loading', 'lazy');
  expect(screen.getByAltText('Детали')).toHaveAttribute('width', '100');
});
it.each([false, true])(
  'opens an accessible dialog and closes it (icon=%s)',
  async (showCloseButton) => {
    const user = userEvent.setup();
    render(
      <Dialog>
        <DialogTrigger>Открыть диалог</DialogTrigger>
        <DialogContent showCloseButton={showCloseButton}>
          <DialogHeader>
            <DialogTitle>Подтверждение</DialogTitle>
            <DialogDescription>Проверьте действие</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose>Отмена</DialogClose>
          </DialogFooter>
        </DialogContent>
      </Dialog>,
    );
    await user.click(screen.getByRole('button', { name: 'Открыть диалог' }));
    const dialog = await screen.findByRole('dialog', { name: 'Подтверждение' });
    expect(dialog).toHaveAccessibleDescription('Проверьте действие');
    await user.click(screen.getByRole('button', { name: showCloseButton ? 'Закрыть' : 'Отмена' }));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  },
);
it.each(['default', 'left', 'top', 'bottom'] as const)(
  'opens and closes a sheet with side=%s',
  async (side) => {
    const user = userEvent.setup();
    render(
      <Sheet>
        <SheetTrigger>Открыть панель</SheetTrigger>
        <SheetContent {...(side === 'default' ? {} : { side })} showCloseButton={side !== 'left'}>
          <SheetHeader>
            <SheetTitle>Панель</SheetTitle>
            <SheetDescription>Дополнительные сведения</SheetDescription>
          </SheetHeader>
          <SheetFooter>
            <SheetClose>Назад</SheetClose>
          </SheetFooter>
        </SheetContent>
      </Sheet>,
    );
    await user.click(screen.getByRole('button', { name: 'Открыть панель' }));
    expect(await screen.findByRole('dialog', { name: 'Панель' })).toHaveAccessibleDescription(
      'Дополнительные сведения',
    );
    await user.click(screen.getByRole('button', { name: side === 'left' ? 'Назад' : 'Close' }));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  },
);
it('shows a tooltip on focus and dismisses it with Escape', async () => {
  const user = userEvent.setup();
  render(
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>Помощь</TooltipTrigger>
        <TooltipContent>Описание кнопки</TooltipContent>
      </Tooltip>
    </TooltipProvider>,
  );
  await user.tab();
  expect(await screen.findByText('Описание кнопки')).toBeInTheDocument();
  await user.keyboard('{Escape}');
  await waitFor(() => expect(screen.queryByText('Описание кнопки')).not.toBeInTheDocument());
});
it('updates mobile detection when the breakpoint changes and cleans up the listener', () => {
  let listener = () => {};
  const remove = vi.fn();
  vi.spyOn(window, 'matchMedia').mockImplementation(
    () =>
      ({
        matches: false,
        addEventListener: (_event: string, callback: EventListenerOrEventListenerObject) => {
          listener = callback as () => void;
        },
        removeEventListener: remove,
      }) as unknown as MediaQueryList,
  );
  vi.spyOn(window, 'innerWidth', 'get').mockReturnValue(1000);
  const hook = renderHook(() => useIsMobile());
  expect(hook.result.current).toBe(false);
  vi.spyOn(window, 'innerWidth', 'get').mockReturnValue(390);
  act(() => listener());
  expect(hook.result.current).toBe(true);
  hook.unmount();
  expect(remove).toHaveBeenCalledWith('change', listener);
});
it('shows request loading, empty and retryable error states', () => {
  const retry = vi.fn();
  const view = render(<RequestState loading />);
  expect(screen.getByRole('status')).toHaveTextContent('Загрузка');
  view.rerender(<RequestState empty="Пустой список" />);
  expect(screen.getByText('Пустой список')).toBeInTheDocument();
  view.rerender(<RequestState error={new Error('Ошибка')} retry={retry} />);
  fireEvent.click(screen.getByRole('button', { name: 'Повторить' }));
  expect(retry).toHaveBeenCalledOnce();
});
it('handles missing names and roles without inventing profile data', () => {
  expect(getUserInitials()).toBe('?');
  expect(getUserInitials(' Анна ')).toBe('А');
  expect(getUserRoleLabel()).toBe('Стажер');
  expect(getUserRoleLabel('mentor')).toBe('Наставник');
  expect(isMentor(null)).toBe(false);
  expect(isTrainee(undefined)).toBe(false);
  expect(isMentor({ ...intern, role: 'mentor' })).toBe(true);
  expect(isTrainee(intern)).toBe(true);
});
it('supports a custom logout callback without changing the session', async () => {
  const user = userEvent.setup(),
    logout = vi.fn();
  mockApi();
  mount(<UserCard user={intern} onLogout={logout} showRole={false} />);
  await user.click(screen.getByRole('button', { name: `Профиль: ${intern.name}` }));
  await user.click(await screen.findByRole('menuitem', { name: 'Выйти' }));
  expect(logout).toHaveBeenCalledOnce();
});
it('shows a disabled profile while loading, then a fallback name without logout', async () => {
  let resolve!: (response: Response) => void;
  mockApi((path) =>
    path === '/api/users'
      ? new Promise<Response>((r) => {
          resolve = r;
        })
      : undefined,
  );
  mount(<UserCard showLogout={false} />);
  expect(screen.getByRole('button')).toBeDisabled();
  await act(async () => resolve(json([])));
  await waitFor(() =>
    expect(screen.getByRole('button', { name: 'Профиль: Пользователь' })).toBeEnabled(),
  );
  await userEvent.setup().click(screen.getByRole('button', { name: 'Профиль: Пользователь' }));
  await screen.findByRole('menu');
  expect(screen.queryByRole('menuitem', { name: 'Выйти' })).not.toBeInTheDocument();
  expect(screen.getByText('Пользователь')).toBeInTheDocument();
});
