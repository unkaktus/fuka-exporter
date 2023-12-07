
typedef struct BinaryInfo
{
  double mass1, mass2;             // Rest masses
  double position_x1, position_x2; // Compact object positions
} BinaryInfo;

typedef enum BinaryType
{
  BNS,
  BBH,
  BHNS,
} BinaryType;

BinaryInfo read_binary_info(BinaryType binary_type, char *info_filename);

typedef struct Grid
{
  double *x, *y, *z;
  int n_points;
} Grid;

typedef struct Fields
{
  double *alpha;
  double *beta_x, *beta_y, *beta_z;
  double *gamma_xx, *gamma_xy, *gamma_xz,
      *gamma_yy, *gamma_yz,
      *gamma_zz;
  double *K_xx, *K_xy, *K_xz,
      *K_yy, *K_yz,
      *K_zz;
  // Hydro variables
  double *rho;
  double *epsilon;
  double *pressure;
  double *v_x, *v_y, *v_z;
} Fields;

void free_fields(Fields *fields);

typedef struct FUKAInterpolateRequest
{
  BinaryType binary_type;
  char *info_filename;
  Grid *grid;
  double interpolation_offset;
  int interpolation_order;
  double relative_dr_spacing;
} FUKAInterpolateRequest;

Fields interpolate_FUKA_ID(FUKAInterpolateRequest *req);
Fields allocate_fields(int n_points);
