// Ana sayfada 2 dakikada bir donen ilham sozleri. Bilerek klise "basari"
// vecizeleri degil; inovatif dusunurlerden, ozgun ve dogru olanlar secildi.
export type Quote = {
  text: string;
  author: string;
};

export const QUOTES: Quote[] = [
  {
    text: 'Geleceği tahmin etmenin en iyi yolu onu icat etmektir.',
    author: 'Alan Kay',
  },
  {
    text: "En tehlikeli cümle şudur: 'Bu hep böyle yapıldı.'",
    author: 'Grace Hopper',
  },
  {
    text: 'Sadelik, nihai incelmişliktir.',
    author: 'Leonardo da Vinci',
  },
  {
    text: 'Bir şeyi değiştirmek istiyorsan, eskisini geçersiz kılan yeni bir model kur.',
    author: 'Buckminster Fuller',
  },
  {
    text: 'Hayatta hiçbir şeyden korkulmaz; her şey yalnızca anlaşılır.',
    author: 'Marie Curie',
  },
];

export const QUOTE_ROTATE_MS = 2 * 60 * 1000;
