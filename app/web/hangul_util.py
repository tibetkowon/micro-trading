"""한글 초성 검색 유틸리티 — 순수 유니코드 연산"""

from __future__ import annotations

# 한글 유니코드 범위
_HANGUL_BASE = 0xAC00  # '가'
_HANGUL_END = 0xD7A3  # '힣'
_CHOSUNG_BASE = 0x3131  # 'ㄱ'

# 초성 21자 (유니코드 순서)
_CHOSUNG_LIST = [
    "ㄱ", "ㄲ", "ㄴ", "ㄷ", "ㄸ", "ㄹ", "ㅁ", "ㅂ", "ㅃ",
    "ㅅ", "ㅆ", "ㅇ", "ㅈ", "ㅉ", "ㅊ", "ㅋ", "ㅌ", "ㅍ", "ㅎ",
]

# 초성 집합 (빠른 판별용)
_CHOSUNG_SET = set(_CHOSUNG_LIST)

# 중성 개수 = 21, 종성 개수 = 28
_JUNGSUNG_COUNT = 21
_JONGSUNG_COUNT = 28


def extract_chosung(text: str) -> str:
    """문자열에서 한글 글자의 초성만 추출한다.

    >>> extract_chosung("삼성전자")
    'ㅅㅅㅈㅈ'
    >>> extract_chosung("LG화학")
    'ㅎㅎ'
    """
    result: list[str] = []
    for ch in text:
        code = ord(ch)
        if _HANGUL_BASE <= code <= _HANGUL_END:
            idx = (code - _HANGUL_BASE) // (_JUNGSUNG_COUNT * _JONGSUNG_COUNT)
            result.append(_CHOSUNG_LIST[idx])
    return "".join(result)


def is_chosung_only(text: str) -> bool:
    """문자열이 초성(자음)만으로 구성되어 있는지 판별한다.

    >>> is_chosung_only("ㅅㅅ")
    True
    >>> is_chosung_only("삼성")
    False
    """
    return bool(text) and all(ch in _CHOSUNG_SET for ch in text)


def match_chosung(query: str, target: str) -> bool:
    """초성 쿼리가 대상 문자열의 초성과 매칭되는지 확인한다.

    부분 매칭을 지원한다 (prefix 또는 substring).

    >>> match_chosung("ㅅㅅ", "삼성전자")
    True
    >>> match_chosung("ㅈㅈ", "삼성전자")
    True
    >>> match_chosung("ㅎㄷ", "현대차")
    True
    """
    target_chosung = extract_chosung(target)
    return query in target_chosung
