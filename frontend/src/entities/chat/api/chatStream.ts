import type { SSEEvent } from '../model/types';

const NO_ANSWER_KEYWORDS = ['наставник', 'нет ответа', 'кто отвечает', 'зарплат'];

const NO_ANSWER_TEXT =
  'В базе знаний нет ответа на этот вопрос, а гадать я не буду. За такие вопросы отвечает ваш наставник — напишите ему, он поможет.';

const EQUATOR_TEXT =
  'Информацию о заполнении экватора вы найдёте в регламенте стажировки. Форма отчёта прикреплена внутри раздела.';

function createDelay(signal: AbortSignal) {
  return (ms: number) =>
    new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(resolve, ms);
      signal.addEventListener('abort', () => {
        clearTimeout(timeout);
        reject(new DOMException('Aborted', 'AbortError'));
      });
    });
}

async function streamWords(
  text: string,
  onEvent: (event: SSEEvent) => void,
  getSeq: () => number,
  delay: (ms: number) => Promise<void>,
) {
  const words = text.split(' ');
  for (let i = 0; i < words.length; i++) {
    const chunk = (i > 0 ? ' ' : '') + words[i];
    onEvent({
      type: 'delta',
      seq: getSeq(),
      delta: chunk,
    });
    await delay(35);
  }
}

export async function simulateSSEStream(
  userText: string,
  onEvent: (event: SSEEvent) => void,
  signal: AbortSignal,
): Promise<void> {
  const isNoAnswer = NO_ANSWER_KEYWORDS.some((kw) => userText.toLowerCase().includes(kw));
  const delay = createDelay(signal);
  let seq = 1;
  const nextSeq = () => seq++;

  try {
    onEvent({
      type: 'stage',
      seq: nextSeq(),
      id: 'parse',
      label: 'Разбираю вопрос',
      status: 'completed',
    });
    await delay(350);

    onEvent({
      type: 'stage',
      seq: nextSeq(),
      id: 'search',
      label: 'Ищу в базе знаний',
      status: 'completed',
    });
    await delay(450);

    onEvent({
      type: 'stage',
      seq: nextSeq(),
      id: 'generate',
      label: 'Формулирую ответ...',
      status: 'in_progress',
    });
    await delay(250);

    if (isNoAnswer) {
      await streamWords(NO_ANSWER_TEXT, onEvent, nextSeq, delay);
      await delay(150);

      onEvent({
        type: 'card',
        seq: nextSeq(),
        cardType: 'mentor_fallback',
        payload: {
          id: 'mentor-1',
          name: 'Мария Соколова',
          role: 'Наставник стажёров',
          team: 'Команда онбординга',
          schedule: 'Отвечает в будни с 10:00 до 19:00',
          initials: 'МС',
          isOnline: true,
        },
      });

      onEvent({
        type: 'done',
        seq: nextSeq(),
        finalText: NO_ANSWER_TEXT,
      });
    } else {
      await streamWords(EQUATOR_TEXT, onEvent, nextSeq, delay);
      await delay(150);

      onEvent({
        type: 'card',
        seq: nextSeq(),
        cardType: 'wiki_source',
        payload: {
          title: 'Памятка стажёра: Материалы к экватору',
          section: 'Раздел 2.3 «Промежуточная аттестация»',
          quote:
            '«...заполните шаблон формы обратной связи и согласуйте результаты спринтов с наставником до конца 4-й недели...»',
          url: '/wiki/materials/equator',
        },
      });

      onEvent({
        type: 'done',
        seq: nextSeq(),
        finalText: EQUATOR_TEXT,
      });
    }
  } catch {
    if (signal.aborted) return;
    onEvent({
      type: 'error',
      seq: nextSeq(),
      error: 'Не удалось получить ответ от сервиса. Попробуйте еще раз.',
      canRetry: true,
    });
  }
}
