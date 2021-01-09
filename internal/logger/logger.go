package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Domain uint8

const (
	UnknownDomain Domain = iota
	AllDomain
	InitDomain
	GraphDomain
	ModuleInfoDomain
	PackageInfoDomain
	ModuleDependencyDomain
	ParserDomain
	QueryDomain
	PrinterDomain
)

func domainFromString(domain string) Domain {
	return map[string]Domain{
		"all":     AllDomain,
		"init":    InitDomain,
		"graph":   GraphDomain,
		"modinfo": ModuleInfoDomain,
		"pkginfo": PackageInfoDomain,
		"moddeps": ModuleDependencyDomain,
		"parser":  ParserDomain,
		"query":   QueryDomain,
		"printer": PrinterDomain,
	}[domain]
}

func stringFromDomain(domain Domain) string {
	return map[Domain]string{
		AllDomain:              "all",
		InitDomain:             "init",
		GraphDomain:            "graph",
		ModuleInfoDomain:       "modinfo",
		PackageInfoDomain:      "pkginfo",
		ModuleDependencyDomain: "moddeps",
		ParserDomain:           "parser",
		QueryDomain:            "query",
		PrinterDomain:          "printer",
	}[domain]
}

type Builder struct {
	log          *zap.Logger
	enc          *goModEncoder
	defaultLevel zapcore.Level
	domainLevels map[Domain]zapcore.Level
	cache        map[Domain]*Logger
}

func NewBuilder(out zapcore.WriteSyncer) *Builder {
	enc := newEncoder()
	return &Builder{
		log:          zap.New(zapcore.NewCore(enc, out, zapcore.DebugLevel)),
		enc:          enc,
		domainLevels: map[Domain]zapcore.Level{},
		cache:        map[Domain]*Logger{},
	}
}

func (b *Builder) SetDomainLevel(domain string, level zapcore.Level) {
	d := domainFromString(domain)
	switch d {
	case UnknownDomain:
		b.log.Warn("Unrecognised logger domain.")
	case AllDomain:
		b.defaultLevel = level
	default:
		b.domainLevels[d] = level
	}
}

func (b *Builder) Log() *Logger {
	return b.logger(AllDomain)
}

func (b *Builder) Domain(domain Domain) *Logger {
	return b.logger(domain)
}

func (b *Builder) logger(domain Domain) *Logger {
	if _, ok := b.cache[domain]; !ok {
		targetLevel := b.defaultLevel
		if lvl, ok := b.domainLevels[domain]; ok {
			targetLevel = lvl
		}
		b.cache[domain] = &Logger{
			Logger: b.log.Named(stringFromDomain(domain)).WithOptions(zap.IncreaseLevel(targetLevel)),
			enc:    b.enc,
		}
	}
	return b.cache[domain]
}

type Logger struct {
	*zap.Logger
	enc *goModEncoder
}

func (l *Logger) AddIndent() {
	l.enc.indent++
}

func (l *Logger) RemoveIndent() {
	if l.enc.indent > 0 {
		l.enc.indent--
	}
}
