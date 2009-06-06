// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.


#define		EXTERN
#include	"go.h"
#include	"y.tab.h"
#include <ar.h>

#define	DBG	if(!debug['x']);else print
enum
{
	EOF		= -1,
};

int
main(int argc, char *argv[])
{
	int c;

	outfile = nil;
	package = "____";
	ARGBEGIN {
	default:
		c = ARGC();
		if(c >= 0 && c < sizeof(debug))
			debug[c]++;
		break;

	case 'o':
		outfile = ARGF();
		break;

	case 'k':
		package = ARGF();
		break;

	case 'I':
		addidir(ARGF());
		break;
	} ARGEND

	if(argc != 1)
		goto usage;

	pathname = mal(100);
	if(getwd(pathname, 99) == 0)
		strcpy(pathname, "/???");

	fmtinstall('O', Oconv);		// node opcodes
	fmtinstall('E', Econv);		// etype opcodes
	fmtinstall('J', Jconv);		// all the node flags
	fmtinstall('S', Sconv);		// sym pointer
	fmtinstall('T', Tconv);		// type pointer
	fmtinstall('N', Nconv);		// node pointer
	fmtinstall('Z', Zconv);		// escaped string
	fmtinstall('L', Lconv);		// line number
	fmtinstall('B', Bconv);		// big numbers
	fmtinstall('F', Fconv);		// big float numbers

	betypeinit();
	if(maxround == 0 || widthptr == 0)
		fatal("betypeinit failed");

	lexinit();
	typeinit(LATYPE);

	lineno = 1;
	block = 1;
	blockgen = 1;

	setfilename(argv[0]);
	infile = argv[0];
	linehist(infile, 0, 0);

	curio.infile = infile;
	curio.bin = Bopen(infile, OREAD);
	if(curio.bin == nil)
		fatal("cant open: %s", infile);
	curio.peekc = 0;
	curio.peekc1 = 0;

	externdcl = mal(sizeof(*externdcl));
	externdcl->back = externdcl;
	dclcontext = PEXTERN;

	exportlist = mal(sizeof(*exportlist));
	exportlist->back = exportlist;

	typelist = mal(sizeof(*typelist));
	typelist->back = typelist;

	// function field skeleton
	fskel = nod(OLIST, N, nod(OLIST, N, N));
	fskel->left = nod(ODCLFIELD, N, N);
	fskel->right->left = nod(ODCLFIELD, N, N);
	fskel->right->right = nod(ODCLFIELD, N, N);

	nerrors = 0;
	yyparse();
	runifacechecks();

	linehist(nil, 0, 0);
	if(curio.bin != nil)
		Bterm(curio.bin);

	if(nerrors)
		errorexit();

	dumpobj();

	if(nerrors)
		errorexit();

	exit(0);
	return 0;

usage:
	print("flags:\n");
	print("  -I DIR search for packages in DIR\n");
	print("  -d print declarations\n");
	print("  -e no limit on number of errors printed\n");
	print("  -f print stack frame structure\n");
	print("  -h panic on an error\n");
	print("  -k name specify package name\n");
	print("  -o file specify output file\n");
	print("  -p print the assembly language\n");
	print("  -w print the parse tree after typing\n");
	print("  -x print lex tokens\n");
	exit(0);
	return 0;
}

void
setfilename(char *file)
{
	char *p;
	int c;

	p = strrchr(file, '/');
	if(p != nil)
		file = p+1;
	strncpy(namebuf, file, sizeof(namebuf));
	p = strrchr(namebuf, '.');
	if(p != nil && strcmp(p, ".go") == 0)
		*p = 0;
	filename = strdup(namebuf);

	// turn invalid identifier chars into _
	for(p=filename; *p; p++) {
		c = *p & 0xFF;
		if(c < 0x80 && !isalpha(c) && !isdigit(c) && c != '_')
			*p = '_';
	}
}

int
arsize(Biobuf *b, char *name)
{
	struct ar_hdr *a;

	if((a = Brdline(b, '\n')) == nil)
		return -1;
	if(Blinelen(b) != sizeof(struct ar_hdr))
		return -1;
	if(strncmp(a->name, name, strlen(name)) != 0)
		return -1;
	return atoi(a->size);
}

int
skiptopkgdef(Biobuf *b)
{
	char *p;
	int sz;

	/* archive header */
	if((p = Brdline(b, '\n')) == nil)
		return 0;
	if(Blinelen(b) != 8)
		return 0;
	if(memcmp(p, "!<arch>\n", 8) != 0)
		return 0;
	/* symbol table is first; skip it */
	sz = arsize(b, "__.SYMDEF");
	if(sz < 0)
		return 0;
	Bseek(b, sz, 1);
	/* package export block is second */
	sz = arsize(b, "__.PKGDEF");
	if(sz <= 0)
		return 0;
	return 1;
}

void
addidir(char* dir)
{
	Idir** pp;

	if(dir == nil)
		return;

	for(pp = &idirs; *pp != nil; pp = &(*pp)->link)
		;
	*pp = mal(sizeof(Idir));
	(*pp)->link = nil;
	(*pp)->dir = dir;
}

int
findpkg(Strlit *name)
{
	static char *goroot, *goos, *goarch;
	Idir *p;

	if(goroot == nil) {
		goroot = getenv("GOROOT");
		goos = getenv("GOOS");
		goarch = getenv("GOARCH");
	}

	// try .a before .6.  important for building libraries:
	// if there is an array.6 in the array.a library,
	// want to find all of array.a, not just array.6.
	for(p = idirs; p != nil; p = p->link) {
		snprint(namebuf, sizeof(namebuf), "%s/%Z.a", p->dir, name);
		if(access(namebuf, 0) >= 0)
			return 1;
		snprint(namebuf, sizeof(namebuf), "%s/%Z.%c", p->dir, name, thechar);
		if(access(namebuf, 0) >= 0)
			return 1;
	}

	snprint(namebuf, sizeof(namebuf), "%Z.a", name);
	if(access(namebuf, 0) >= 0)
		return 1;
	snprint(namebuf, sizeof(namebuf), "%Z.%c", name, thechar);
	if(access(namebuf, 0) >= 0)
		return 1;
	if(goroot != nil) {
		snprint(namebuf, sizeof(namebuf), "%s/pkg/%s_%s/%Z.a", goroot, goos, goarch, name);
		if(access(namebuf, 0) >= 0)
			return 1;
		snprint(namebuf, sizeof(namebuf), "%s/pkg/%s_%s/%Z.%c", goroot, goos, goarch, name, thechar);
		if(access(namebuf, 0) >= 0)
			return 1;
	}
	return 0;
}

void
importfile(Val *f)
{
	Biobuf *imp;
	char *file, *p;
	int32 c;
	int len;

	if(f->ctype != CTSTR) {
		yyerror("import statement not a string");
		return;
	}

	if(strcmp(f->u.sval->s, "unsafe") == 0) {
		cannedimports("unsafe.6", unsafeimport);
		return;
	}

	if(!findpkg(f->u.sval))
		fatal("can't find import: %Z", f->u.sval);
	imp = Bopen(namebuf, OREAD);
	if(imp == nil)
		fatal("can't open import: %Z", f->u.sval);
	file = strdup(namebuf);

	len = strlen(namebuf);
	if(len > 2 && namebuf[len-2] == '.' && namebuf[len-1] == 'a') {
		if(!skiptopkgdef(imp))
			fatal("import not package file: %s", namebuf);

		// assume .a files move (get installed)
		// so don't record the full path.
		p = file + len - f->u.sval->len - 2;
		linehist(p, 0, 0);
		linehist(p, -1, 1);	// acts as #pragma lib
	} else {
		// assume .6 files don't move around
		// so do record the full path
		linehist(file, 0, 0);
		linehist(file, -1, 0);
	}

	/*
	 * position the input right
	 * after $$ and return
	 */
	pushedio = curio;
	curio.bin = imp;
	curio.peekc = 0;
	curio.peekc1 = 0;
	curio.infile = file;
	for(;;) {
		c = getc();
		if(c == EOF)
			break;
		if(c != '$')
			continue;
		c = getc();
		if(c == EOF)
			break;
		if(c != '$')
			continue;
		return;
	}
	yyerror("no import in: %Z", f->u.sval);
	unimportfile();
}

void
unimportfile(void)
{
	linehist(nil, 0, 0);

	if(curio.bin != nil) {
		Bterm(curio.bin);
		curio.bin = nil;
	} else
		lineno--;	// re correct sys.6 line number
	curio = pushedio;
	pushedio.bin = nil;
	inimportsys = 0;
}

void
cannedimports(char *file, char *cp)
{
	lineno++;		// if sys.6 is included on line 1,
	linehist(file, 0, 0);	// the debugger gets confused

	pushedio = curio;
	curio.bin = nil;
	curio.peekc = 0;
	curio.peekc1 = 0;
	curio.infile = file;
	curio.cp = cp;

	pkgmyname = S;
	inimportsys = 1;
}

int
isfrog(int c)
{
	// complain about possibly invisible control characters
	if(c < 0)
		return 1;
	if(c < ' ') {
		if(c == '\n' || c== '\r' || c == '\t')	// good white space
			return 0;
		return 1;
	}
	if(0x80 <= c && c <= 0xa0)	// unicode block including unbreakable space.
		return 1;
	return 0;
}

int32
yylex(void)
{
	int c, c1, clen;
	vlong v;
	char *cp;
	Rune rune;
	int escflag;
	Sym *s;

	prevlineno = lineno;

l0:
	c = getc();
	if(isspace(c))
		goto l0;

	if(c >= Runeself) {
		/* all multibyte runes are alpha */
		cp = lexbuf;
		goto talph;
	}

	if(isalpha(c)) {
		cp = lexbuf;
		goto talph;
	}

	if(isdigit(c))
		goto tnum;

	switch(c) {
	case EOF:
		ungetc(EOF);
		return -1;

	case '_':
		cp = lexbuf;
		goto talph;

	case '.':
		c1 = getc();
		if(isdigit(c1)) {
			cp = lexbuf;
			*cp++ = c;
			c = c1;
			c1 = 0;
			goto casedot;
		}
		if(c1 == '.') {
			c1 = getc();
			if(c1 == '.') {
				c = LDDD;
				goto lx;
			}
			ungetc(c1);
			c1 = '.';
		}
		break;

	case '"':
		/* "..." */
		strcpy(lexbuf, "\"<string>\"");
		cp = mal(sizeof(int32));
		clen = sizeof(int32);

	caseq:
		for(;;) {
			if(escchar('"', &escflag, &v))
				break;
			if(v < Runeself || escflag) {
				cp = remal(cp, clen, 1);
				cp[clen++] = v;
			} else {
				// botch - this limits size of runes
				rune = v;
				c = runelen(rune);
				cp = remal(cp, clen, c);
				runetochar(cp+clen, &rune);
				clen += c;
			}
		}
		goto catem;

	case '`':
		/* `...` */
		strcpy(lexbuf, "`<string>`");
		cp = mal(sizeof(int32));
		clen = sizeof(int32);

	casebq:
		for(;;) {
			c = getc();
			if(c == EOF || c == '`')
				break;
			cp = remal(cp, clen, 1);
			cp[clen++] = c;
		}
		goto catem;

	catem:
		c = getc();
		if(isspace(c))
			goto catem;

		// skip comments
		if(c == '/') {
			c1 = getc();
			if(c1 == '*') {
				for(;;) {
					c = getr();
					while(c == '*') {
						c = getr();
						if(c == '/')
							goto catem;
					}
					if(c == EOF) {
						yyerror("eof in comment");
						errorexit();
					}
				}
			}
			if(c1 == '/') {
				for(;;) {
					c = getr();
					if(c == '\n')
						goto catem;
					if(c == EOF) {
						yyerror("eof in comment");
						errorexit();
					}
				}
			}
			ungetc(c1);
		}

		// cat adjacent strings
		if(c == '"')
			goto caseq;
		if(c == '`')
			goto casebq;
		ungetc(c);

		*(int32*)cp = clen-sizeof(int32);	// length
		do {
			cp = remal(cp, clen, 1);
			cp[clen++] = 0;
		} while(clen & MAXALIGN);
		yylval.val.u.sval = (Strlit*)cp;
		yylval.val.ctype = CTSTR;
		DBG("lex: string literal\n");
		return LLITERAL;

	case '\'':
		/* '.' */
		if(escchar('\'', &escflag, &v))
			v = '\'';	// allow '''
		if(!escchar('\'', &escflag, &v)) {
			yyerror("missing '");
			ungetc(v);
		}
		yylval.val.u.xval = mal(sizeof(*yylval.val.u.xval));
		mpmovecfix(yylval.val.u.xval, v);
		yylval.val.ctype = CTINT;
		DBG("lex: codepoint literal\n");
		return LLITERAL;

	case '/':
		c1 = getc();
		if(c1 == '*') {
			for(;;) {
				c = getr();
				while(c == '*') {
					c = getr();
					if(c == '/')
						goto l0;
				}
				if(c == EOF) {
					yyerror("eof in comment");
					errorexit();
				}
			}
		}
		if(c1 == '/') {
			for(;;) {
				c = getr();
				if(c == '\n')
					goto l0;
				if(c == EOF) {
					yyerror("eof in comment");
					errorexit();
				}
			}
		}
		if(c1 == '=') {
			c = ODIV;
			goto asop;
		}
		break;

	case ':':
		c1 = getc();
		if(c1 == '=') {
			c = LCOLAS;
			goto lx;
		}
		break;

	case '*':
		c1 = getc();
		if(c1 == '=') {
			c = OMUL;
			goto asop;
		}
		break;

	case '%':
		c1 = getc();
		if(c1 == '=') {
			c = OMOD;
			goto asop;
		}
		break;

	case '+':
		c1 = getc();
		if(c1 == '+') {
			c = LINC;
			goto lx;
		}
		if(c1 == '=') {
			c = OADD;
			goto asop;
		}
		break;

	case '-':
		c1 = getc();
		if(c1 == '-') {
			c = LDEC;
			goto lx;
		}
		if(c1 == '=') {
			c = OSUB;
			goto asop;
		}
		break;

	case '>':
		c1 = getc();
		if(c1 == '>') {
			c = LRSH;
			c1 = getc();
			if(c1 == '=') {
				c = ORSH;
				goto asop;
			}
			break;
		}
		if(c1 == '=') {
			c = LGE;
			goto lx;
		}
		c = LGT;
		break;

	case '<':
		c1 = getc();
		if(c1 == '<') {
			c = LLSH;
			c1 = getc();
			if(c1 == '=') {
				c = OLSH;
				goto asop;
			}
			break;
		}
		if(c1 == '=') {
			c = LLE;
			goto lx;
		}
		if(c1 == '-') {
			c = LCOMM;
			goto lx;
		}
		c = LLT;
		break;

	case '=':
		c1 = getc();
		if(c1 == '=') {
			c = LEQ;
			goto lx;
		}
		break;

	case '!':
		c1 = getc();
		if(c1 == '=') {
			c = LNE;
			goto lx;
		}
		break;

	case '&':
		c1 = getc();
		if(c1 == '&') {
			c = LANDAND;
			goto lx;
		}
		if(c1 == '^') {
			c = LANDNOT;
			c1 = getc();
			if(c1 == '=') {
				c = OANDNOT;
				goto asop;
			}
			break;
		}
		if(c1 == '=') {
			c = OAND;
			goto asop;
		}
		break;

	case '|':
		c1 = getc();
		if(c1 == '|') {
			c = LOROR;
			goto lx;
		}
		if(c1 == '=') {
			c = OOR;
			goto asop;
		}
		break;

	case '^':
		c1 = getc();
		if(c1 == '=') {
			c = OXOR;
			goto asop;
		}
		break;

	default:
		goto lx;
	}
	ungetc(c1);

lx:
	if(c > 0xff)
		DBG("%L lex: TOKEN %s\n", lineno, lexname(c));
	else
		DBG("%L lex: TOKEN '%c'\n", lineno, c);
	if(isfrog(c)) {
		yyerror("illegal character 0x%ux", c);
		goto l0;
	}
	return c;

asop:
	yylval.lint = c;	// rathole to hold which asop
	DBG("lex: TOKEN ASOP %c\n", c);
	return LASOP;

talph:
	/*
	 * cp is set to lexbuf and some
	 * prefix has been stored
	 */
	for(;;) {
		if(c >= Runeself) {
			for(c1=0;;) {
				cp[c1++] = c;
				if(fullrune(cp, c1)) {
					chartorune(&rune, cp);
					 if(isfrog(rune)) {
					 	yyerror("illegal character 0x%ux", rune);
					 	goto l0;
					 }
					break;
				}
				c = getc();
			}
			cp += c1;
			c = getc();
			continue;
		}
		if(!isalnum(c) && c != '_')
			break;
		*cp++ = c;
		c = getc();
	}
	*cp = 0;
	ungetc(c);

	s = lookup(lexbuf);
	if(s->lexical == LIGNORE)
		goto l0;

	if(context != nil) {
		s = pkglookup(s->name, context);
		if(s->lexical == LIGNORE)
			goto l0;
		if(!exportname(s->name) && strcmp(package, s->package) != 0)
			s = pkglookup(s->name, ".private");
	}

	DBG("lex: %S %s\n", s, lexname(s->lexical));
	yylval.sym = s;
	return s->lexical;

tnum:
	c1 = 0;
	cp = lexbuf;
	if(c != '0') {
		for(;;) {
			*cp++ = c;
			c = getc();
			if(isdigit(c))
				continue;
			goto dc;
		}
	}
	*cp++ = c;
	c = getc();
	if(c == 'x' || c == 'X') {
		for(;;) {
			*cp++ = c;
			c = getc();
			if(isdigit(c))
				continue;
			if(c >= 'a' && c <= 'f')
				continue;
			if(c >= 'A' && c <= 'F')
				continue;
			if(cp == lexbuf+2)
				yyerror("malformed hex constant");
			goto ncu;
		}
	}

	if(c == 'p')	// 0p begins floating point zero
		goto casep;

	c1 = 0;
	for(;;) {
		if(!isdigit(c))
			break;
		if(c < '0' || c > '7')
			c1 = 1;		// not octal
		*cp++ = c;
		c = getc();
	}
	if(c == '.')
		goto casedot;
	if(c == 'e' || c == 'E')
		goto casee;
	if(c1)
		yyerror("malformed octal constant");
	goto ncu;

dc:
	if(c == '.')
		goto casedot;
	if(c == 'e' || c == 'E')
		goto casee;
	if(c == 'p' || c == 'P')
		goto casep;

ncu:
	*cp = 0;
	ungetc(c);

	yylval.val.u.xval = mal(sizeof(*yylval.val.u.xval));
	mpatofix(yylval.val.u.xval, lexbuf);
	if(yylval.val.u.xval->ovf) {
		yyerror("overflow in constant");
		mpmovecfix(yylval.val.u.xval, 0);
	}
	yylval.val.ctype = CTINT;
	DBG("lex: integer literal\n");
	return LLITERAL;

casedot:
	for(;;) {
		*cp++ = c;
		c = getc();
		if(!isdigit(c))
			break;
	}
	if(c != 'e' && c != 'E')
		goto caseout;

casee:
	*cp++ = 'e';
	c = getc();
	if(c == '+' || c == '-') {
		*cp++ = c;
		c = getc();
	}
	if(!isdigit(c))
		yyerror("malformed fp constant exponent");
	while(isdigit(c)) {
		*cp++ = c;
		c = getc();
	}
	goto caseout;

casep:
	*cp++ = 'p';
	c = getc();
	if(c == '+' || c == '-') {
		*cp++ = c;
		c = getc();
	}
	if(!isdigit(c))
		yyerror("malformed fp constant exponent");
	while(isdigit(c)) {
		*cp++ = c;
		c = getc();
	}
	goto caseout;

caseout:
	*cp = 0;
	ungetc(c);

	yylval.val.u.fval = mal(sizeof(*yylval.val.u.fval));
	mpatoflt(yylval.val.u.fval, lexbuf);
	if(yylval.val.u.fval->val.ovf) {
		yyerror("overflow in float constant");
		mpmovecflt(yylval.val.u.fval, 0.0);
	}
	yylval.val.ctype = CTFLT;
	DBG("lex: floating literal\n");
	return LLITERAL;
}

int
getc(void)
{
	int c;

	c = curio.peekc;
	if(c != 0) {
		curio.peekc = curio.peekc1;
		curio.peekc1 = 0;
		if(c == '\n')
			lineno++;
		return c;
	}

	if(curio.bin == nil) {
		c = *curio.cp & 0xff;
		if(c != 0)
			curio.cp++;
	} else
		c = Bgetc(curio.bin);

	switch(c) {
	case 0:
		if(curio.bin != nil)
			break;
	case EOF:
		return EOF;

	case '\n':
		lineno++;
		break;
	}
	return c;
}

void
ungetc(int c)
{
	curio.peekc1 = curio.peekc;
	curio.peekc = c;
	if(c == '\n')
		lineno--;
}

int32
getr(void)
{
	int c, i;
	char str[UTFmax+1];
	Rune rune;

	c = getc();
	if(c < Runeself)
		return c;
	i = 0;
	str[i++] = c;

loop:
	c = getc();
	str[i++] = c;
	if(!fullrune(str, i))
		goto loop;
	c = chartorune(&rune, str);
	if(rune == Runeerror && c == 1) {
		yyerror("illegal rune in string");
		for(c=0; c<i; c++)
			print(" %.2x", *(uchar*)(str+c));
		print("\n");
	}
	return rune;
}

int
getnsc(void)
{
	int c;

	c = getc();
	for(;;) {
		if(!isspace(c))
			return c;
		if(c == '\n') {
			lineno++;
			return c;
		}
		c = getc();
	}
	return 0;
}


int
escchar(int e, int *escflg, vlong *val)
{
	int i, c;
	vlong l;

	*escflg = 0;

loop:
	c = getr();
	if(c == '\n') {
		yyerror("newline in string");
		return 1;
	}
	if(c != '\\') {
		if(c == e)
			return 1;
		*val = c;
		return 0;
	}

	c = getr();
	switch(c) {
	case '\n':
		goto loop;

	case 'x':
		*escflg = 1;	// it's a byte
		i = 2;
		goto hex;

	case 'u':
		i = 4;
		goto hex;

	case 'U':
		i = 8;
		goto hex;

	case '0':
	case '1':
	case '2':
	case '3':
	case '4':
	case '5':
	case '6':
	case '7':
		*escflg = 1;	// it's a byte
		goto oct;

	case 'a': c = '\a'; break;
	case 'b': c = '\b'; break;
	case 'f': c = '\f'; break;
	case 'n': c = '\n'; break;
	case 'r': c = '\r'; break;
	case 't': c = '\t'; break;
	case 'v': c = '\v'; break;
	case '\\': c = '\\'; break;

	default:
		if(c != e)
			yyerror("unknown escape sequence: %c", c);
	}
	*val = c;
	return 0;

hex:
	l = 0;
	for(; i>0; i--) {
		c = getc();
		if(c >= '0' && c <= '9') {
			l = l*16 + c-'0';
			continue;
		}
		if(c >= 'a' && c <= 'f') {
			l = l*16 + c-'a' + 10;
			continue;
		}
		if(c >= 'A' && c <= 'F') {
			l = l*16 + c-'A' + 10;
			continue;
		}
		yyerror("non-hex character in escape sequence: %c", c);
		ungetc(c);
		break;
	}
	*val = l;
	return 0;

oct:
	l = c - '0';
	for(i=2; i>0; i--) {
		c = getc();
		if(c >= '0' && c <= '7') {
			l = l*8 + c-'0';
			continue;
		}
		yyerror("non-oct character in escape sequence: %c", c);
		ungetc(c);
	}
	if(l > 255)
		yyerror("oct escape value > 255: %d", l);

	*val = l;
	return 0;
}

static	struct
{
	char*	name;
	int	lexical;
	int	etype;
} syms[] =
{
/*	name		lexical		etype
 */
/* basic types */
	"int8",		LATYPE,	TINT8,
	"int16",	LATYPE,	TINT16,
	"int32",	LATYPE,	TINT32,
	"int64",	LATYPE,	TINT64,

	"uint8",	LATYPE,	TUINT8,
	"uint16",	LATYPE,	TUINT16,
	"uint32",	LATYPE,	TUINT32,
	"uint64",	LATYPE,	TUINT64,

	"float32",	LATYPE,	TFLOAT32,
	"float64",	LATYPE,	TFLOAT64,
	"float80",	LATYPE,	TFLOAT80,

	"bool",		LATYPE,	TBOOL,
	"byte",		LATYPE,	TUINT8,
	"string",	LATYPE,	TSTRING,

	"any",		LATYPE,	TANY,

	"break",	LBREAK,		Txxx,
	"case",		LCASE,		Txxx,
	"chan",		LCHAN,		Txxx,
	"const",	LCONST,		Txxx,
	"continue",	LCONTINUE,	Txxx,
	"default",	LDEFAULT,	Txxx,
	"else",		LELSE,		Txxx,
	"defer",	LDEFER,		Txxx,
	"fallthrough",	LFALL,		Txxx,
	"false",	LFALSE,		Txxx,
	"for",		LFOR,		Txxx,
	"func",		LFUNC,		Txxx,
	"go",		LGO,		Txxx,
	"goto",		LGOTO,		Txxx,
	"if",		LIF,		Txxx,
	"import",	LIMPORT,	Txxx,
	"interface",	LINTERFACE,	Txxx,
	"iota",		LIOTA,		Txxx,
	"make",		LMAKE,		Txxx,
	"map",		LMAP,		Txxx,
	"new",		LNEW,		Txxx,
	"len",		LLEN,		Txxx,
	"cap",		LCAP,		Txxx,
	"nil",		LNIL,		Txxx,
	"package",	LPACKAGE,	Txxx,
	"panic",	LPANIC,		Txxx,
	"panicln",	LPANICN,	Txxx,
	"print",	LPRINT,		Txxx,
	"println",	LPRINTN,	Txxx,
	"range",	LRANGE,		Txxx,
	"return",	LRETURN,	Txxx,
	"select",	LSELECT,	Txxx,
	"struct",	LSTRUCT,	Txxx,
	"switch",	LSWITCH,	Txxx,
	"true",		LTRUE,		Txxx,
	"type",		LTYPE,		Txxx,
	"var",		LVAR,		Txxx,

	"close",	LCLOSE,		Txxx,
	"closed",	LCLOSED,	Txxx,

	"notwithstanding",		LIGNORE,	Txxx,
	"thetruthofthematter",		LIGNORE,	Txxx,
	"despiteallobjections",		LIGNORE,	Txxx,
	"whereas",			LIGNORE,	Txxx,
	"insofaras",			LIGNORE,	Txxx,
};

void
lexinit(void)
{
	int i, lex;
	Sym *s;
	Type *t;
	int etype;

	/*
	 * initialize basic types array
	 * initialize known symbols
	 */
	for(i=0; i<nelem(syms); i++) {
		lex = syms[i].lexical;
		s = lookup(syms[i].name);
		s->lexical = lex;
		s->package = package;

		etype = syms[i].etype;
		if(etype == Txxx)
			continue;

		if(etype < 0 || etype >= nelem(types))
			fatal("lexinit: %s bad etype", s->name);

		t = types[etype];
		if(t != T) {
			s->otype = t;
			continue;
		}
		t = typ(etype);
		t->sym = s;

		dowidth(t);
		types[etype] = t;
		s->otype = t;
	}

	// logically, the type of a string literal.
	// types[TSTRING] is the named type string
	// (the type of x in var x string or var x = "hello").
	// this is the ideal form
	// (the type of x in const x = "hello").
	// TODO(rsc): this may need some more thought.
	idealstring = typ(TSTRING);
}

struct
{
	int	lex;
	char*	name;
} lexn[] =
{
	LANDAND,	"ANDAND",
	LASOP,		"ASOP",
	LBREAK,		"BREAK",
	LCASE,		"CASE",
	LCHAN,		"CHAN",
	LCOLAS,		"COLAS",
	LCONST,		"CONST",
	LCONTINUE,	"CONTINUE",
	LDEC,		"DEC",
	LELSE,		"ELSE",
	LEQ,		"EQ",
	LFUNC,		"FUNC",
	LGE,		"GE",
	LGO,		"GO",
	LGOTO,		"GOTO",
	LGT,		"GT",
	LIF,		"IF",
	LINC,		"INC",
	LINTERFACE,	"INTERFACE",
	LLE,		"LE",
	LLITERAL,	"LITERAL",
	LLSH,		"LSH",
	LLT,		"LT",
	LMAP,		"MAP",
	LNAME,		"NAME",
	LNE,		"NE",
	LOROR,		"OROR",
	LPACK,		"PACK",
	LRANGE,		"RANGE",
	LRETURN,	"RETURN",
	LRSH,		"RSH",
	LSTRUCT,	"STRUCT",
	LSWITCH,	"SWITCH",
	LTYPE,		"TYPE",
	LVAR,		"VAR",
	LFOR,		"FOR",
	LNEW,		"NEW",
	LLEN,		"LEN",
	LFALL,		"FALL",
	LIOTA,		"IOTA",
	LPRINT,		"PRINT",
	LPACKAGE,	"PACKAGE",
	LIMPORT,	"IMPORT",
	LDEFER,		"DEFER",
	LPANIC,		"PANIC",
};

char*
lexname(int lex)
{
	int i;
	static char buf[100];

	for(i=0; i<nelem(lexn); i++)
		if(lexn[i].lex == lex)
			return lexn[i].name;
	snprint(buf, sizeof(buf), "LEX-%d", lex);
	return buf;
}

void
mkpackage(char* pkg)
{
	Sym *s;
	int32 h;
	char *p;

	if(bout != nil) {
		yyerror("mkpackage: called again %s %s", pkg, package);
		return;
	}

	// redefine all names to be this package
	for(h=0; h<NHASH; h++)
		for(s = hash[h]; s != S; s = s->link)
			if(s->package == package)
				s->package = pkg;
	package = pkg;

	// declare this name as a package
	s = lookup(package);
	s->lexical = LPACK;
	s->opack = s->name;

	if(outfile == nil) {
		p = strrchr(infile, '/');
		if(p == nil)
			p = infile;
		else
			p = p+1;
		snprint(namebuf, sizeof(namebuf), "%s", p);
		p = strrchr(namebuf, '.');
		if(p != nil)
			*p = 0;
		outfile = smprint("%s.%c", namebuf, thechar);
	}
}
