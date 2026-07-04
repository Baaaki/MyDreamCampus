// Ana sayfada 2 dakikada bir donen ilham sozleri. Bilerek klise "basari"
// vecizeleri degil; inovatif dusunurlerden ozgun secimler + insanlarin ortak
// dusunme hatalarini (kendini kandirma, yanlis kesinlik, asiri guven) vurgulayanlar.
export type Quote = {
  text: string;
  author: string;
};

export const QUOTES: Quote[] = [
  { text: 'Zamanın sınırlı; onu başkasının hayatını yaşayarak harcama.', author: 'Steve Jobs' },
  { text: 'Yaratıcılık, aslında şeyleri birbirine bağlamaktır.', author: 'Steve Jobs' },
  {
    text: 'Kandırmaman gereken ilk kişi kendinsin; kandırması en kolay kişi de sensin.',
    author: 'Richard Feynman',
  },
  { text: 'Yaratamadığım şeyi gerçekten anlamış sayılmam.', author: 'Richard Feynman' },
  { text: 'Bir şeyin adını bilmek, onu bilmek değildir.', author: 'Richard Feynman' },
  { text: 'Geleceği tahmin etmenin en iyi yolu onu icat etmektir.', author: 'Alan Kay' },
  { text: "En tehlikeli cümle şudur: 'Bu hep böyle yapıldı.'", author: 'Grace Hopper' },
  { text: 'Sadelik, nihai incelmişliktir.', author: 'Leonardo da Vinci' },
  {
    text: 'Değiştirmek istiyorsan, eskisini geçersiz kılan yeni bir model kur.',
    author: 'Buckminster Fuller',
  },
  { text: 'Hayatta hiçbir şeyden korkulmaz; her şey yalnızca anlaşılır.', author: 'Marie Curie' },
  {
    text: 'Başını derde sokan, bilmediklerin değil; kesin doğru sandığın yanlışlardır.',
    author: 'Mark Twain',
  },
  { text: 'Olağanüstü iddialar, olağanüstü kanıtlar gerektirir.', author: 'Carl Sagan' },
  { text: 'Bir fikre duyduğun güven, onun doğru olduğunu göstermez.', author: 'Daniel Kahneman' },
  {
    text: 'Hiç yapılmaması gereken bir işi verimli yapmak kadar boşa çaba yoktur.',
    author: 'Peter Drucker',
  },
  {
    text: 'Çoğu insan, pes ettiğinde başarıya ne kadar yaklaştığını fark etmez.',
    author: 'Thomas Edison',
  },
  {
    text: 'Kimse senin iznin olmadan sana kendini değersiz hissettiremez.',
    author: 'Eleanor Roosevelt',
  },
  { text: 'Yalnız kalmayı öğren; icadın sırrı budur.', author: 'Nikola Tesla' },
  {
    text: 'Gemi yaptırmak istiyorsan, önce insanlara denizin özlemini aşıla.',
    author: 'Antoine de Saint-Exupéry',
  },
  { text: 'Önemli olan ne kadar yavaş gittiğin değil, durmadığındır.', author: 'Konfüçyüs' },
  {
    text: 'İster yapabileceğine inan ister yapamayacağına; her hâlükârda haklısın.',
    author: 'Henry Ford',
  },
];

export const QUOTE_ROTATE_MS = 2 * 60 * 1000;
