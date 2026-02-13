# Memory & Performance Optimization Skill

## Python Runtime
- **Generators**: 대량의 데이터를 처리할 때는 리스트 대신 제너레이터(yield)를 사용한다.
- **Resource Cleanup**: 파일이나 네트워크 연결은 반드시 `with` 문이나 `try-finally`로 닫는다.
- **Object Recycling**: 루프 안에서 불필요한 객체 생성을 최소화한다.

## FastAPI & Networking
- **Keep-Alive**: API 연결 시 커넥션 풀을 적절히 관리하여 오버헤드를 줄인다.
- **Payload Size**: 외부 API 응답에서 필요한 필드만 추출하여 메모리에 올린다.

## SQLite Optimization
- **cache_size**: `-4000` (4MB). 1GB RAM 환경에서 8MB는 과다하므로 4MB로 제한.
- **mmap_size**: `67108864` (64MB). 가상 주소 공간만 사용하므로 물리 RAM 부담 없음.
- **temp_store=MEMORY**: 임시 테이블을 디스크 대신 메모리에 생성하여 I/O 절감.
- **journal_mode=WAL**: 읽기/쓰기 동시 가능. 단, WAL 파일이 커지지 않도록 주기적 checkpoint 고려.

## Swap Memory Usage
- **Swap Awareness**: 2GB Swap이 있지만, 가급적 물리 RAM(1GB) 내에서 동작하도록 설계한다. Swap은 예외적인 상황(피크 타임)을 위한 안전장치로 간주한다.
